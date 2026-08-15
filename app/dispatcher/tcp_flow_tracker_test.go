// SPDX-License-Identifier: MPL-2.0

package dispatcher

import (
	"context"
	"testing"

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
	assertFlowSnapshot(t, active[0], routing.TCPFlowActive)
	if active[0].ClosedAt.IsZero() == false {
		t.Fatal("active flow has a close timestamp")
	}

	close(release)
	<-done
	closed := dispatcher.SnapshotTCPFlows()
	if len(closed) != 1 {
		t.Fatalf("closed snapshot count = %d, want 1", len(closed))
	}
	assertFlowSnapshot(t, closed[0], routing.TCPFlowClosed)
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
	finish()

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

func assertFlowSnapshot(t *testing.T, flow routing.TCPFlowSnapshot, state routing.TCPFlowState) {
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
	if flow.OutboundTag != "proxy-a" {
		t.Fatalf("outbound tag = %q", flow.OutboundTag)
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
	if flow.StartedAt.IsZero() {
		t.Fatal("flow has no start timestamp")
	}
}

type flowTestHandler struct {
	tag     string
	started chan struct{}
	release chan struct{}
}

func (h *flowTestHandler) Tag() string { return h.tag }

func (h *flowTestHandler) Dispatch(_ context.Context, link *transport.Link) {
	mb, _ := link.Reader.ReadMultiBuffer()
	mb = buf.ReleaseMulti(mb)
	_ = link.Writer.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("download"))})
	if h.started != nil {
		close(h.started)
	}
	if h.release != nil {
		<-h.release
	}
}

func (*flowTestHandler) SenderSettings() *serial.TypedMessage { return nil }
func (*flowTestHandler) ProxySettings() *serial.TypedMessage  { return nil }
func (*flowTestHandler) Start() error                         { return nil }
func (*flowTestHandler) Close() error                         { return nil }

type flowTestOutboundManager struct {
	handler outbound.Handler
}

func (*flowTestOutboundManager) Type() interface{} { return outbound.ManagerType() }
func (*flowTestOutboundManager) Start() error      { return nil }
func (*flowTestOutboundManager) Close() error      { return nil }
func (m *flowTestOutboundManager) GetHandler(tag string) outbound.Handler {
	if m.handler.Tag() == tag {
		return m.handler
	}
	return nil
}
func (m *flowTestOutboundManager) GetDefaultHandler() outbound.Handler { return m.handler }
func (*flowTestOutboundManager) AddHandler(context.Context, outbound.Handler) error {
	return nil
}
func (*flowTestOutboundManager) RemoveHandler(context.Context, string) error { return nil }
func (m *flowTestOutboundManager) ListHandlers(context.Context) []outbound.Handler {
	return []outbound.Handler{m.handler}
}
