// SPDX-License-Identifier: MPL-2.0

package dispatcher

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/pipe"
)

func TestRoutedDispatchTracksTCPFlowLifecycle(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handler := &flowTestHandler{
		tag:     "proxy-a",
		started: started,
		release: release,
	}
	dispatcher := &DefaultDispatcher{
		ohm:      &flowTestOutboundManager{handler: handler},
		tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
	}
	dispatcher.EnableTCPFlowTracking()

	input := &buf.MultiBufferContainer{MultiBuffer: buf.MultiBuffer{buf.FromBytes([]byte("upload"))}}
	output := new(buf.MultiBufferContainer)
	link := &transport.Link{Reader: input, Writer: output}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source: net.TCPDestination(net.IPAddress([]byte{10, 0, 0, 2}), 41000),
	})

	done := make(chan struct{})
	go func() {
		dispatcher.routedDispatch(ctx, link, destination)
		close(done)
	}()
	<-started

	active := dispatcher.SnapshotTCPFlows()
	if len(active) != 1 {
		t.Fatalf("active snapshot count = %d, want 1", len(active))
	}
	assertFlowSnapshot(t, active[0], routing.TCPFlowActive, "")
	if active[0].ClosedAt.IsZero() == false {
		t.Fatal("active flow has a close timestamp")
	}

	close(release)
	<-done
	closed := dispatcher.SnapshotTCPFlows()
	if len(closed) != 1 {
		t.Fatalf("closed snapshot count = %d, want 1", len(closed))
	}
	assertFlowSnapshot(t, closed[0], routing.TCPFlowClosed, routing.TCPFlowCompleted)
	if closed[0].ClosedAt.IsZero() {
		t.Fatal("closed flow has no close timestamp")
	}
	if got := output.MultiBuffer.String(); got != "download" {
		t.Fatalf("downlink payload = %q, want %q", got, "download")
	}
	output.MultiBuffer = buf.ReleaseMulti(output.MultiBuffer)
}

func TestTCPFlowTrackerPreservesNativePipeTypes(t *testing.T) {
	uplinkReader, uplinkWriter := pipe.New()
	downlinkReader, downlinkWriter := pipe.New()
	tracker := newTCPFlowTracker(tcpFlowHistoryLimit)

	tracked, finish := tracker.track(&transport.Link{
		Reader: uplinkReader,
		Writer: downlinkWriter,
	}, "", "tcp:example.com:443", "proxy-a")
	if tracked.Reader != uplinkReader {
		t.Fatalf("native pipe reader was replaced with %T", tracked.Reader)
	}
	if tracked.Writer != downlinkWriter {
		t.Fatalf("native pipe writer was replaced with %T", tracked.Writer)
	}
	if _, ok := tracked.Reader.(buf.TimeoutReader); !ok {
		t.Fatal("native pipe reader lost buf.TimeoutReader")
	}

	if err := uplinkWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}); err != nil {
		t.Fatal(err)
	}
	uplink, err := tracked.Reader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	uplink = buf.ReleaseMulti(uplink)
	if err := tracked.Writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("download"))}); err != nil {
		t.Fatal(err)
	}
	downlink, err := downlinkReader.ReadMultiBuffer()
	if err != nil {
		t.Fatal(err)
	}
	downlink = buf.ReleaseMulti(downlink)
	finish(routing.TCPFlowCompleted)
	finish(routing.TCPFlowFailed)

	flows := tracker.snapshot()
	if len(flows) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(flows))
	}
	if flows[0].UplinkBytes != int64(len("upload")) {
		t.Fatalf("uplink bytes = %d", flows[0].UplinkBytes)
	}
	if flows[0].DownlinkBytes != int64(len("download")) {
		t.Fatalf("downlink bytes = %d", flows[0].DownlinkBytes)
	}
	if flows[0].EndReason != routing.TCPFlowCompleted {
		t.Fatalf("end reason = %q, want %q", flows[0].EndReason, routing.TCPFlowCompleted)
	}
}

func TestRoutedDispatchMarksTCPFlowFailedAndForwardsError(t *testing.T) {
	handlerErr := errors.New("handler failed")
	handler := &flowTestHandler{tag: "proxy-a", fail: handlerErr}
	dispatcher := newFlowTestDispatcher(handler)
	collector := new(flowTestErrorCollector)
	ctx, link, output, destination := newFlowTestRequest(context.Background())
	ctx = session.TrackedConnectionError(ctx, collector)

	dispatcher.routedDispatch(ctx, link, destination)

	assertClosedFlow(t, dispatcher, routing.TCPFlowFailed)
	if !errors.Is(collector.err, handlerErr) {
		t.Fatalf("forwarded error = %v, want %v", collector.err, handlerErr)
	}
	if collector.count != 1 {
		t.Fatalf("forwarded error count = %d, want 1", collector.count)
	}
	releaseFlowTestOutput(output)
}

func TestRoutedDispatchMarksTCPFlowCancelled(t *testing.T) {
	started := make(chan struct{})
	handler := &flowTestHandler{tag: "proxy-a", started: started, waitForCancel: true}
	dispatcher := newFlowTestDispatcher(handler)
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx, link, output, destination := newFlowTestRequest(baseCtx)
	done := make(chan struct{})

	go func() {
		dispatcher.routedDispatch(ctx, link, destination)
		close(done)
	}()
	<-started
	active := dispatcher.SnapshotTCPFlows()
	if len(active) != 1 {
		t.Fatalf("active snapshot count = %d, want 1", len(active))
	}
	assertFlowSnapshot(t, active[0], routing.TCPFlowActive, "")

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after context cancellation")
	}

	assertClosedFlow(t, dispatcher, routing.TCPFlowCancelled)
	releaseFlowTestOutput(output)
}

func TestTCPFlowFailureTakesPrecedenceOverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	feedback := &tcpFlowFeedback{parent: ctx}
	feedback.SubmitError(errors.New("handler failed"))

	if got := feedback.endReason(ctx); got != routing.TCPFlowFailed {
		t.Fatalf("end reason = %q, want %q", got, routing.TCPFlowFailed)
	}
}

func TestRoutedDispatchTracksFinalOutboundSelection(t *testing.T) {
	tests := []struct {
		name      string
		routeTag  string
		forcedTag string
		wantTag   string
	}{
		{name: "default", wantTag: "default"},
		{name: "routed", routeTag: "routed", wantTag: "routed"},
		{name: "forced takes precedence", routeTag: "routed", forcedTag: "forced", wantTag: "forced"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handlers := map[string]outbound.Handler{
				"default": &flowTestHandler{tag: "default"},
				"routed":  &flowTestHandler{tag: "routed"},
				"forced":  &flowTestHandler{tag: "forced"},
			}
			dispatcher := &DefaultDispatcher{
				ohm: &flowTestOutboundManager{
					handlers:       handlers,
					defaultHandler: handlers["default"],
				},
				tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
			}
			if test.routeTag == "" {
				dispatcher.router = &flowTestRouter{err: errors.New("no matching route")}
			} else {
				dispatcher.router = &flowTestRouter{route: &flowTestRoute{
					outboundTag: test.routeTag,
					ruleTag:     "test-rule",
				}}
			}
			dispatcher.EnableTCPFlowTracking()

			ctx, link, output, destination := newFlowTestRequest(context.Background())
			if test.forcedTag != "" {
				ctx = session.SetForcedOutboundTagToContext(ctx, test.forcedTag)
			}
			dispatcher.routedDispatch(ctx, link, destination)

			flows := dispatcher.SnapshotTCPFlows()
			if len(flows) != 1 {
				t.Fatalf("snapshot count = %d, want exactly one closed flow", len(flows))
			}
			assertFlowSnapshot(t, flows[0], routing.TCPFlowClosed, routing.TCPFlowCompleted, test.wantTag)
			if got := session.OutboundsFromContext(ctx)[0].Tag; got != test.wantTag {
				t.Fatalf("session outbound tag = %q, want %q", got, test.wantTag)
			}
			releaseFlowTestOutput(output)
		})
	}
}

func TestRoutedDispatchDoesNotTrackMissingDetour(t *testing.T) {
	tests := []struct {
		name      string
		routeTag  string
		forcedTag string
	}{
		{name: "missing routed handler", routeTag: "missing"},
		{name: "missing forced handler", forcedTag: "missing"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defaultHandler := &flowTestHandler{tag: "default"}
			dispatcher := &DefaultDispatcher{
				ohm: &flowTestOutboundManager{
					handlers:       map[string]outbound.Handler{"default": defaultHandler},
					defaultHandler: defaultHandler,
				},
				tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
			}
			if test.routeTag != "" {
				dispatcher.router = &flowTestRouter{route: &flowTestRoute{outboundTag: test.routeTag}}
			}
			dispatcher.EnableTCPFlowTracking()

			ctx, link, uplinkWriter, downlinkWriter, destination := newFlowTestPipeRequest(context.Background())
			if test.forcedTag != "" {
				ctx = session.SetForcedOutboundTagToContext(ctx, test.forcedTag)
			}
			dispatcher.routedDispatch(ctx, link, destination)

			if flows := dispatcher.SnapshotTCPFlows(); len(flows) != 0 {
				t.Fatalf("missing detour produced %d flow snapshots", len(flows))
			}
			assertPipeWriterClosed(t, uplinkWriter)
			assertPipeWriterClosed(t, downlinkWriter)
		})
	}
}

func TestRoutedDispatchWaitsForSubmittedLifecycle(t *testing.T) {
	lifecycleResult := make(chan error, 1)
	handoffDone := make(chan struct{})
	handler := &flowTestHandler{
		tag:             "mux-out",
		started:         handoffDone,
		lifecycleResult: lifecycleResult,
	}
	dispatcher := newFlowTestDispatcher(handler)
	ctx, link, output, destination := newFlowTestRequest(context.Background())
	done := make(chan struct{})
	go func() {
		dispatcher.routedDispatch(ctx, link, destination)
		close(done)
	}()

	select {
	case <-handoffDone:
	case <-time.After(time.Second):
		t.Fatal("handler did not submit its asynchronous lifecycle")
	}
	active := dispatcher.SnapshotTCPFlows()
	if len(active) != 1 {
		t.Fatalf("active snapshot count = %d, want 1", len(active))
	}
	assertFlowSnapshot(t, active[0], routing.TCPFlowActive, "", "mux-out")
	select {
	case <-done:
		t.Fatal("dispatcher flow closed at asynchronous handoff")
	default:
	}

	lifecycleResult <- nil
	close(lifecycleResult)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher flow did not close after submitted lifecycle")
	}
	flows := dispatcher.SnapshotTCPFlows()
	if len(flows) != 1 {
		t.Fatalf("snapshot count = %d, want one closed flow", len(flows))
	}
	assertFlowSnapshot(t, flows[0], routing.TCPFlowClosed, routing.TCPFlowCompleted, "mux-out")
	releaseFlowTestOutput(output)
}

func TestRoutedDispatchMarksSubmittedLifecycleFailure(t *testing.T) {
	lifecycleErr := errors.New("asynchronous outbound failed")
	lifecycleResult := make(chan error, 1)
	handoffDone := make(chan struct{})
	handler := &flowTestHandler{
		tag:             "mux-out",
		started:         handoffDone,
		lifecycleResult: lifecycleResult,
	}
	dispatcher := newFlowTestDispatcher(handler)
	collector := new(flowTestErrorCollector)
	ctx, link, output, destination := newFlowTestRequest(context.Background())
	ctx = session.TrackedConnectionError(ctx, collector)
	done := make(chan struct{})
	go func() {
		dispatcher.routedDispatch(ctx, link, destination)
		close(done)
	}()
	<-handoffDone

	lifecycleResult <- lifecycleErr
	close(lifecycleResult)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher flow did not close after failed lifecycle result")
	}
	assertClosedFlow(t, dispatcher, routing.TCPFlowFailed, "mux-out")
	if !errors.Is(collector.err, lifecycleErr) {
		t.Fatalf("forwarded lifecycle error = %v, want %v", collector.err, lifecycleErr)
	}
	if collector.count != 1 {
		t.Fatalf("forwarded lifecycle error count = %d, want 1", collector.count)
	}
	releaseFlowTestOutput(output)
}

func TestRoutedDispatchMarksSubmittedLifecycleCancellation(t *testing.T) {
	lifecycleResult := make(chan error, 1)
	handoffDone := make(chan struct{})
	handler := &flowTestHandler{
		tag:             "mux-out",
		started:         handoffDone,
		lifecycleResult: lifecycleResult,
	}
	dispatcher := newFlowTestDispatcher(handler)
	collector := new(flowTestErrorCollector)
	ctx, link, output, destination := newFlowTestRequest(context.Background())
	ctx = session.TrackedConnectionError(ctx, collector)
	done := make(chan struct{})
	go func() {
		dispatcher.routedDispatch(ctx, link, destination)
		close(done)
	}()
	<-handoffDone

	lifecycleResult <- context.Canceled
	close(lifecycleResult)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("dispatcher flow did not close after cancelled lifecycle result")
	}
	assertClosedFlow(t, dispatcher, routing.TCPFlowCancelled, "mux-out")
	if collector.count != 0 {
		t.Fatalf("forwarded cancellation error count = %d, want 0", collector.count)
	}
	releaseFlowTestOutput(output)
}

func TestRoutedDispatchDoesNotTrackTCPUntilEnabled(t *testing.T) {
	handler := &flowTestHandler{tag: "proxy-a"}
	dispatcher := &DefaultDispatcher{
		ohm:      &flowTestOutboundManager{handler: handler},
		tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
	}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	link := &transport.Link{
		Reader: &buf.MultiBufferContainer{},
		Writer: &buf.MultiBufferContainer{},
	}
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})

	dispatcher.routedDispatch(ctx, link, destination)
	if flows := dispatcher.SnapshotTCPFlows(); len(flows) != 0 {
		t.Fatalf("disabled tracker produced %d snapshots", len(flows))
	}
}

func TestRoutedDispatchDoesNotTrackUDP(t *testing.T) {
	handler := &flowTestHandler{tag: "proxy-a"}
	dispatcher := &DefaultDispatcher{
		ohm:      &flowTestOutboundManager{handler: handler},
		tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
	}
	dispatcher.EnableTCPFlowTracking()
	link := &transport.Link{
		Reader: &buf.MultiBufferContainer{},
		Writer: &buf.MultiBufferContainer{},
	}
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{}})

	dispatcher.routedDispatch(ctx, link, net.UDPDestination(net.DomainAddress("example.com"), 53))
	if flows := dispatcher.SnapshotTCPFlows(); len(flows) != 0 {
		t.Fatalf("UDP produced %d TCP flow snapshots", len(flows))
	}
}

func assertFlowSnapshot(t *testing.T, flow routing.TCPFlowSnapshot, state routing.TCPFlowState, reason routing.TCPFlowEndReason, outboundTag ...string) {
	t.Helper()
	if flow.FlowID == 0 {
		t.Fatal("FlowID is zero")
	}
	if flow.Source != "tcp:10.0.0.2:41000" {
		t.Fatalf("source = %q", flow.Source)
	}
	if flow.Destination != "tcp:example.com:443" {
		t.Fatalf("destination = %q", flow.Destination)
	}
	wantOutboundTag := "proxy-a"
	if len(outboundTag) != 0 {
		wantOutboundTag = outboundTag[0]
	}
	if flow.OutboundTag != wantOutboundTag {
		t.Fatalf("outbound tag = %q, want %q", flow.OutboundTag, wantOutboundTag)
	}
	if flow.UplinkBytes != int64(len("upload")) {
		t.Fatalf("uplink bytes = %d", flow.UplinkBytes)
	}
	if flow.DownlinkBytes != int64(len("download")) {
		t.Fatalf("downlink bytes = %d", flow.DownlinkBytes)
	}
	if flow.State != state {
		t.Fatalf("state = %q, want %q", flow.State, state)
	}
	if flow.EndReason != reason {
		t.Fatalf("end reason = %q, want %q", flow.EndReason, reason)
	}
	if flow.StartedAt.IsZero() {
		t.Fatal("flow has no start timestamp")
	}
}

func newFlowTestDispatcher(handler outbound.Handler) *DefaultDispatcher {
	dispatcher := &DefaultDispatcher{
		ohm:      &flowTestOutboundManager{handler: handler},
		tcpFlows: newTCPFlowTracker(tcpFlowHistoryLimit),
	}
	dispatcher.EnableTCPFlowTracking()
	return dispatcher
}

func newFlowTestRequest(ctx context.Context) (context.Context, *transport.Link, *buf.MultiBufferContainer, net.Destination) {
	input := &buf.MultiBufferContainer{MultiBuffer: buf.MultiBuffer{buf.FromBytes([]byte("upload"))}}
	output := new(buf.MultiBufferContainer)
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx = session.ContextWithOutbounds(ctx, []*session.Outbound{{Target: destination}})
	ctx = session.ContextWithInbound(ctx, &session.Inbound{
		Source: net.TCPDestination(net.IPAddress([]byte{10, 0, 0, 2}), 41000),
	})
	return ctx, &transport.Link{Reader: input, Writer: output}, output, destination
}

func newFlowTestPipeRequest(ctx context.Context) (context.Context, *transport.Link, *pipe.Writer, *pipe.Writer, net.Destination) {
	uplinkReader, uplinkWriter := pipe.New()
	_, downlinkWriter := pipe.New()
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx = session.ContextWithOutbounds(ctx, []*session.Outbound{{Target: destination}})
	return ctx, &transport.Link{Reader: uplinkReader, Writer: downlinkWriter}, uplinkWriter, downlinkWriter, destination
}

func assertPipeWriterClosed(t *testing.T, writer *pipe.Writer) {
	t.Helper()
	err := writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("closed-check"))})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error = %v, want %v", err, io.ErrClosedPipe)
	}
}

func assertClosedFlow(t *testing.T, dispatcher *DefaultDispatcher, reason routing.TCPFlowEndReason, outboundTag ...string) {
	t.Helper()
	flows := dispatcher.SnapshotTCPFlows()
	if len(flows) != 1 {
		t.Fatalf("snapshot count = %d, want exactly one closed flow", len(flows))
	}
	assertFlowSnapshot(t, flows[0], routing.TCPFlowClosed, reason, outboundTag...)
	if flows[0].ClosedAt.IsZero() {
		t.Fatal("closed flow has no close timestamp")
	}
}

func releaseFlowTestOutput(output *buf.MultiBufferContainer) {
	output.MultiBuffer = buf.ReleaseMulti(output.MultiBuffer)
}

type flowTestHandler struct {
	tag             string
	started         chan struct{}
	release         chan struct{}
	fail            error
	waitForCancel   bool
	lifecycleResult <-chan error
}

func (h *flowTestHandler) Tag() string { return h.tag }

func (h *flowTestHandler) Dispatch(ctx context.Context, link *transport.Link) {
	mb, _ := link.Reader.ReadMultiBuffer()
	mb = buf.ReleaseMulti(mb)
	_ = link.Writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("download"))})
	if h.started != nil {
		close(h.started)
	}
	if h.lifecycleResult != nil {
		session.SubmitOutboundLifecycleToOriginator(ctx, h.lifecycleResult)
		return
	}
	if h.fail != nil {
		session.SubmitOutboundErrorToOriginator(ctx, h.fail)
		return
	}
	if h.waitForCancel {
		<-ctx.Done()
		return
	}
	if h.release != nil {
		<-h.release
	}
}

type flowTestErrorCollector struct {
	err   error
	count int
}

func (c *flowTestErrorCollector) SubmitError(err error) {
	c.err = err
	c.count++
}

func (*flowTestHandler) SenderSettings() *serial.TypedMessage { return nil }
func (*flowTestHandler) ProxySettings() *serial.TypedMessage  { return nil }
func (*flowTestHandler) Start() error                         { return nil }
func (*flowTestHandler) Close() error                         { return nil }

type flowTestOutboundManager struct {
	handler        outbound.Handler
	handlers       map[string]outbound.Handler
	defaultHandler outbound.Handler
}

func (*flowTestOutboundManager) Type() interface{} { return outbound.ManagerType() }
func (*flowTestOutboundManager) Start() error      { return nil }
func (*flowTestOutboundManager) Close() error      { return nil }
func (m *flowTestOutboundManager) GetHandler(tag string) outbound.Handler {
	if m.handlers != nil {
		return m.handlers[tag]
	}
	if m.handler != nil && m.handler.Tag() == tag {
		return m.handler
	}
	return nil
}
func (m *flowTestOutboundManager) GetDefaultHandler() outbound.Handler {
	if m.defaultHandler != nil {
		return m.defaultHandler
	}
	return m.handler
}
func (*flowTestOutboundManager) AddHandler(context.Context, outbound.Handler) error {
	return nil
}
func (*flowTestOutboundManager) RemoveHandler(context.Context, string) error { return nil }
func (m *flowTestOutboundManager) ListHandlers(context.Context) []outbound.Handler {
	if m.handlers != nil {
		handlers := make([]outbound.Handler, 0, len(m.handlers))
		for _, handler := range m.handlers {
			handlers = append(handlers, handler)
		}
		return handlers
	}
	return []outbound.Handler{m.handler}
}

type flowTestRouter struct {
	routing.DefaultRouter
	route routing.Route
	err   error
}

func (r *flowTestRouter) PickRoute(routing.Context) (routing.Route, error) {
	return r.route, r.err
}

type flowTestRoute struct {
	routing.Context
	outboundTag string
	ruleTag     string
}

func (*flowTestRoute) GetOutboundGroupTags() []string { return nil }
func (r *flowTestRoute) GetOutboundTag() string       { return r.outboundTag }
func (r *flowTestRoute) GetRuleTag() string           { return r.ruleTag }
