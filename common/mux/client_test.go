package mux_test

import (
	"context"
	goerrors "errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/mux"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/testing/mocks"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/pipe"
)

func TestIncrementalPickerFailure(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	mockWorkerFactory := mocks.NewMuxClientWorkerFactory(mockCtl)
	mockWorkerFactory.EXPECT().Create().Return(nil, errors.New("test"))

	picker := mux.IncrementalWorkerPicker{
		Factory: mockWorkerFactory,
	}

	_, err := picker.PickAvailable()
	if err == nil {
		t.Error("expected error, but nil")
	}
}

func TestClientWorkerEOF(t *testing.T) {
	reader, writer := pipe.New(pipe.WithoutSizeLimit())
	common.Must(writer.Close())

	worker, err := mux.NewClientWorker(transport.Link{Reader: reader, Writer: writer}, mux.ClientStrategy{})
	common.Must(err)

	time.Sleep(time.Millisecond * 500)

	f := worker.Dispatch(context.Background(), nil)
	if f {
		t.Error("expected failed dispatching, but actually not")
	}
}

func TestClientWorkerClose(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	r1, w1 := pipe.New(pipe.WithoutSizeLimit())
	worker1, err := mux.NewClientWorker(transport.Link{
		Reader: r1,
		Writer: w1,
	}, mux.ClientStrategy{
		MaxConcurrency: 4,
		MaxConnection:  4,
	})
	common.Must(err)

	r2, w2 := pipe.New(pipe.WithoutSizeLimit())
	worker2, err := mux.NewClientWorker(transport.Link{
		Reader: r2,
		Writer: w2,
	}, mux.ClientStrategy{
		MaxConcurrency: 4,
		MaxConnection:  4,
	})
	common.Must(err)

	factory := mocks.NewMuxClientWorkerFactory(mockCtl)
	gomock.InOrder(
		factory.EXPECT().Create().Return(worker1, nil),
		factory.EXPECT().Create().Return(worker2, nil),
	)

	picker := &mux.IncrementalWorkerPicker{
		Factory: factory,
	}
	manager := &mux.ClientManager{
		Picker: picker,
	}

	tr1, tw1 := pipe.New(pipe.WithoutSizeLimit())
	ctx1 := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{
		Target: net.TCPDestination(net.DomainAddress("www.example.com"), 80),
	}})
	common.Must(manager.Dispatch(ctx1, &transport.Link{
		Reader: tr1,
		Writer: tw1,
	}))
	defer tw1.Close()

	common.Must(w1.Close())

	time.Sleep(time.Millisecond * 500)
	if !worker1.Closed() {
		t.Error("worker1 is not finished")
	}

	tr2, tw2 := pipe.New(pipe.WithoutSizeLimit())
	ctx2 := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{
		Target: net.TCPDestination(net.DomainAddress("www.example.com"), 80),
	}})
	common.Must(manager.Dispatch(ctx2, &transport.Link{
		Reader: tr2,
		Writer: tw2,
	}))
	defer tw2.Close()

	common.Must(w2.Close())
}

func TestClientWorkerPublishesLogicalSessionLifecycle(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	muxUplinkReader, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		common.Close(muxDownlinkWriter)
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	outputReader, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}

	var lifecycleResult <-chan error
	select {
	case lifecycleResult = <-feedback.results:
	case <-time.After(time.Second):
		t.Fatal("mux worker did not publish the logical session lifecycle")
	}
	select {
	case <-lifecycleResult:
		t.Fatal("logical session lifecycle closed at native-pipe handoff")
	default:
	}
	request, err := muxUplinkReader.ReadMultiBuffer()
	common.Must(err)
	request = buf.ReleaseMulti(request)

	response := mux.NewResponseWriter(1, muxDownlinkWriter, protocol.TransferTypeStream)
	common.Must(response.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("download"))}))
	common.Must(response.Close())
	select {
	case result := <-lifecycleResult:
		if result != nil {
			t.Fatalf("logical session result = %v, want normal completion", result)
		}
	case <-time.After(time.Second):
		t.Fatal("logical session lifecycle did not close after remote end")
	}

	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
	if worker.Closed() {
		t.Fatal("physical mux connection closed with the logical session")
	}
	if err := inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("after-close"))}); !goerrors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("logical input write error = %v, want %v", err, io.ErrClosedPipe)
	}
	output, err := outputReader.ReadMultiBuffer()
	common.Must(err)
	if got := output.String(); got != "download" {
		t.Fatalf("logical downlink payload = %q, want %q", got, "download")
	}
	output = buf.ReleaseMulti(output)
}

func TestClientWorkerCloseCancelsActiveLogicalSession(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	muxUplinkReader, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() { common.Close(muxDownlinkWriter) })

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}

	var lifecycleResult <-chan error
	select {
	case lifecycleResult = <-feedback.results:
	case <-time.After(time.Second):
		t.Fatal("mux worker did not publish the logical session lifecycle")
	}
	request, err := muxUplinkReader.ReadMultiBuffer()
	common.Must(err)
	request = buf.ReleaseMulti(request)
	common.Must(worker.Close())
	select {
	case result := <-lifecycleResult:
		if !goerrors.Is(result, context.Canceled) {
			t.Fatalf("active logical session result = %v, want context cancellation", result)
		}
	case <-time.After(time.Second):
		t.Fatal("logical session lifecycle did not close with the physical worker")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerRemoteErrorFailsLogicalSession(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	muxUplinkReader, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		common.Close(muxDownlinkWriter)
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}

	lifecycleResult := <-feedback.results
	request, err := muxUplinkReader.ReadMultiBuffer()
	common.Must(err)
	request = buf.ReleaseMulti(request)
	frame := buf.New()
	meta := mux.FrameMetadata{SessionID: 1, SessionStatus: mux.SessionStatusEnd}
	meta.Option.Set(mux.OptionError)
	common.Must(meta.WriteTo(frame))
	common.Must(muxDownlinkWriter.WriteMultiBuffer(buf.MultiBuffer{frame}))

	select {
	case result := <-lifecycleResult:
		if result == nil {
			t.Fatal("remote OptionError completed the logical session normally")
		}
	case <-time.After(time.Second):
		t.Fatal("logical session lifecycle did not close after remote error")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
	if worker.Closed() {
		t.Fatal("remote logical error closed the physical mux worker")
	}
}

func TestClientWorkerPhysicalEOFFailsAllActiveLogicalSessions(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	_, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() { common.Close(worker) })

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 2)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputWriters := make([]*pipe.Writer, 0, 2)
	for range 2 {
		inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
		_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
		inputWriters = append(inputWriters, inputWriter)
		common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
		if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
			t.Fatal("mux worker rejected the logical session")
		}
	}
	t.Cleanup(func() {
		for _, writer := range inputWriters {
			common.Close(writer)
		}
	})
	results := []<-chan error{<-feedback.results, <-feedback.results}
	if worker.ActiveConnections() != 2 {
		t.Fatalf("active mux sessions before physical EOF = %d, want 2", worker.ActiveConnections())
	}
	for i, resultChannel := range results {
		select {
		case result := <-resultChannel:
			t.Fatalf("logical session %d ended before physical EOF with result %v", i+1, result)
		default:
		}
	}
	common.Must(muxDownlinkWriter.Close())

	for i, resultChannel := range results {
		select {
		case result := <-resultChannel:
			if result == nil {
				t.Fatalf("logical session %d completed normally after physical EOF", i+1)
			}
			if goerrors.Is(result, context.Canceled) {
				t.Fatalf("logical session %d was cancelled instead of failed after physical EOF", i+1)
			}
		case <-time.After(time.Second):
			t.Fatalf("logical session %d did not close after physical EOF", i+1)
		}
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerEndWriteFailureFailsLogicalSession(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	uplinkWriter := &endRejectingWriter{err: goerrors.New("end write failed")}
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: uplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		common.Close(muxDownlinkWriter)
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}

	lifecycleResult := <-feedback.results
	common.Must(inputWriter.Close())
	select {
	case result := <-lifecycleResult:
		if result == nil {
			t.Fatal("failed mux End write completed the logical session normally")
		}
		if goerrors.Is(result, context.Canceled) {
			t.Fatalf("failed mux End write cancelled the logical session: %v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("logical session did not close after failed mux End write")
	}
	if uplinkWriter.endCount != 1 {
		t.Fatalf("mux End write attempts = %d, want 1", uplinkWriter.endCount)
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerRemoteEndWithDataCompletesAfterPayload(t *testing.T) {
	muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
	muxUplinkReader, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: muxDownlinkReader,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		common.Close(muxDownlinkWriter)
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}
	t.Cleanup(func() { common.Close(inputWriter) })
	lifecycleResult := <-feedback.results
	request, err := muxUplinkReader.ReadMultiBuffer()
	common.Must(err)
	request = buf.ReleaseMulti(request)

	frame := buf.New()
	meta := mux.FrameMetadata{SessionID: 1, SessionStatus: mux.SessionStatusEnd}
	meta.Option.Set(mux.OptionData)
	common.Must(meta.WriteTo(frame))
	common.Must2(serial.WriteUint16(frame, 4))
	common.Must2(frame.Write([]byte("tail")))
	common.Must(muxDownlinkWriter.WriteMultiBuffer(buf.MultiBuffer{frame}))

	select {
	case result := <-lifecycleResult:
		if result != nil {
			t.Fatalf("complete remote End payload result = %v, want normal completion", result)
		}
	case <-time.After(time.Second):
		t.Fatal("logical session did not close after complete remote End payload")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerRemoteEndReservationRejectsTruncatedPayload(t *testing.T) {
	downlink := newStagedEndReader()
	muxUplinkReader, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: downlink,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() { common.Close(worker) })

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}
	lifecycleResult := <-feedback.results
	request, err := muxUplinkReader.ReadMultiBuffer()
	common.Must(err)
	request = buf.ReleaseMulti(request)

	frame := buf.New()
	meta := mux.FrameMetadata{SessionID: 1, SessionStatus: mux.SessionStatusEnd}
	meta.Option.Set(mux.OptionData)
	common.Must(meta.WriteTo(frame))
	common.Must2(serial.WriteUint16(frame, 2))
	common.Must2(frame.Write([]byte("x")))
	downlink.frames <- buf.MultiBuffer{frame}
	<-downlink.payloadReadStarted

	common.Must(inputWriter.Close())
	endFrame := make(chan buf.MultiBuffer, 1)
	endReadErr := make(chan error, 1)
	go func() {
		mb, err := muxUplinkReader.ReadMultiBuffer()
		if err != nil {
			endReadErr <- err
			return
		}
		endFrame <- mb
	}()
	select {
	case err := <-endReadErr:
		t.Fatalf("reading local End frame: %v", err)
	case mb := <-endFrame:
		container := &buf.MultiBufferContainer{MultiBuffer: mb}
		var endMeta mux.FrameMetadata
		common.Must(endMeta.Unmarshal(container, false))
		common.Close(container)
		if endMeta.SessionStatus != mux.SessionStatusEnd {
			t.Fatalf("local terminal metadata = %+v, want End", endMeta)
		}
	case <-time.After(time.Second):
		t.Fatal("local input EOF did not attempt a mux End frame")
	}
	select {
	case result := <-lifecycleResult:
		t.Fatalf("logical session ended before remote payload validation: %v", result)
	default:
	}

	downlink.release()
	select {
	case result := <-lifecycleResult:
		if result == nil {
			t.Fatal("truncated remote End payload completed the logical session normally")
		}
		if goerrors.Is(result, context.Canceled) {
			t.Fatalf("truncated remote End payload cancelled the logical session: %v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("logical session did not fail after truncated remote End payload")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerCancellationClosesStalledRemoteEnd(t *testing.T) {
	downlink := newStagedEndReader()
	_, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: downlink,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		downlink.release()
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	baseCtx, cancel := context.WithCancel(context.Background())
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(baseCtx, []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}
	t.Cleanup(func() { common.Close(inputWriter) })
	lifecycleResult := <-feedback.results

	frame := buf.New()
	meta := mux.FrameMetadata{SessionID: 1, SessionStatus: mux.SessionStatusEnd}
	meta.Option.Set(mux.OptionData)
	common.Must(meta.WriteTo(frame))
	common.Must2(serial.WriteUint16(frame, 2))
	common.Must2(frame.Write([]byte("x")))
	downlink.frames <- buf.MultiBuffer{frame}
	<-downlink.payloadReadStarted

	cancel()
	select {
	case result := <-lifecycleResult:
		if !goerrors.Is(result, context.Canceled) {
			t.Fatalf("stalled remote End cancellation result = %v, want context cancellation", result)
		}
	case <-time.After(time.Second):
		t.Fatal("context cancellation did not close stalled remote End session")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions after cancellation = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerRemoteErrorPrecedesStalledPayloadCancellation(t *testing.T) {
	downlink := newStagedEndReader()
	_, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
	worker, err := mux.NewClientWorker(transport.Link{
		Reader: downlink,
		Writer: muxUplinkWriter,
	}, mux.ClientStrategy{MaxConcurrency: 4, MaxConnection: 4})
	common.Must(err)
	t.Cleanup(func() {
		downlink.release()
		common.Close(worker)
	})

	feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
	baseCtx, cancel := context.WithCancel(context.Background())
	destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
	ctx := session.ContextWithOutbounds(baseCtx, []*session.Outbound{{Target: destination}})
	ctx = session.TrackedConnectionError(ctx, feedback)
	inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
	_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
	common.Must(inputWriter.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes([]byte("upload"))}))
	if !worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter}) {
		t.Fatal("mux worker rejected the logical session")
	}
	t.Cleanup(func() { common.Close(inputWriter) })
	lifecycleResult := <-feedback.results

	frame := buf.New()
	meta := mux.FrameMetadata{SessionID: 1, SessionStatus: mux.SessionStatusEnd}
	meta.Option.Set(mux.OptionError)
	meta.Option.Set(mux.OptionData)
	common.Must(meta.WriteTo(frame))
	common.Must2(serial.WriteUint16(frame, 2))
	common.Must2(frame.Write([]byte("x")))
	downlink.frames <- buf.MultiBuffer{frame}
	<-downlink.payloadReadStarted
	cancel()

	select {
	case result := <-lifecycleResult:
		if result == nil {
			t.Fatal("remote OptionError with stalled payload completed normally")
		}
		if goerrors.Is(result, context.Canceled) {
			t.Fatalf("remote OptionError lost failure precedence to cancellation: %v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("remote OptionError did not close stalled payload session")
	}
	if worker.ActiveConnections() != 0 {
		t.Fatalf("active mux sessions after remote OptionError = %d, want 0", worker.ActiveConnections())
	}
}

func TestClientWorkerConcurrentDispatchAndClose(t *testing.T) {
	for range 100 {
		muxDownlinkReader, muxDownlinkWriter := pipe.New(pipe.WithoutSizeLimit())
		_, muxUplinkWriter := pipe.New(pipe.WithoutSizeLimit())
		worker, err := mux.NewClientWorker(transport.Link{
			Reader: muxDownlinkReader,
			Writer: muxUplinkWriter,
		}, mux.ClientStrategy{MaxConcurrency: 1, MaxConnection: 1})
		common.Must(err)

		feedback := &muxLifecycleFeedback{results: make(chan (<-chan error), 1)}
		destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
		ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
		ctx = session.TrackedConnectionError(ctx, feedback)
		inputReader, inputWriter := pipe.New(pipe.WithoutSizeLimit())
		_, outputWriter := pipe.New(pipe.WithoutSizeLimit())
		start := make(chan struct{})
		closed := make(chan struct{})
		go func() {
			<-start
			common.Must(worker.Close())
			close(closed)
		}()
		close(start)
		accepted := worker.Dispatch(ctx, &transport.Link{Reader: inputReader, Writer: outputWriter})
		<-closed

		if accepted {
			var resultChannel <-chan error
			select {
			case resultChannel = <-feedback.results:
			case <-time.After(time.Second):
				t.Fatal("accepted session did not publish its lifecycle")
			}
			select {
			case result := <-resultChannel:
				if !goerrors.Is(result, context.Canceled) {
					t.Fatalf("accepted session result = %v, want context cancellation", result)
				}
			case <-time.After(time.Second):
				t.Fatal("accepted session did not close with its worker")
			}
		}
		common.Close(inputWriter)
		common.Close(muxDownlinkWriter)
		if worker.ActiveConnections() != 0 {
			t.Fatalf("active mux sessions = %d, want 0", worker.ActiveConnections())
		}
	}
}

type muxLifecycleFeedback struct {
	results chan (<-chan error)
}

type endRejectingWriter struct {
	err      error
	endCount int
}

type stagedEndReader struct {
	frames             chan buf.MultiBuffer
	payloadReadStarted chan struct{}
	releasePayload     chan struct{}
	frameRead          bool
	releaseOnce        sync.Once
}

func newStagedEndReader() *stagedEndReader {
	return &stagedEndReader{
		frames:             make(chan buf.MultiBuffer, 1),
		payloadReadStarted: make(chan struct{}),
		releasePayload:     make(chan struct{}),
	}
}

func (r *stagedEndReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	if !r.frameRead {
		r.frameRead = true
		select {
		case frame := <-r.frames:
			return frame, nil
		case <-r.releasePayload:
			return nil, io.EOF
		}
	}
	close(r.payloadReadStarted)
	<-r.releasePayload
	return nil, io.EOF
}

func (r *stagedEndReader) Interrupt() {
	r.release()
}

func (r *stagedEndReader) release() {
	r.releaseOnce.Do(func() { close(r.releasePayload) })
}

func (w *endRejectingWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	container := &buf.MultiBufferContainer{MultiBuffer: mb}
	defer container.Close()
	var meta mux.FrameMetadata
	if err := meta.Unmarshal(container, false); err != nil {
		return err
	}
	if meta.SessionStatus == mux.SessionStatusEnd {
		w.endCount++
		return w.err
	}
	return nil
}

func (*muxLifecycleFeedback) SubmitError(error) {}

func (f *muxLifecycleFeedback) SubmitLifecycle(result <-chan error) {
	f.results <- result
}
