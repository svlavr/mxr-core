package mux

import (
	goerrors "errors"
	"testing"

	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/protocol"
)

type rejectingWriter struct {
	err  error
	meta FrameMetadata
}

func (w *rejectingWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	container := &buf.MultiBufferContainer{MultiBuffer: mb}
	defer container.Close()
	if err := w.meta.Unmarshal(container, false); err != nil {
		return err
	}
	return w.err
}

func TestWriterCloseReturnsEndWriteError(t *testing.T) {
	wantErr := goerrors.New("write rejected")
	output := &rejectingWriter{err: wantErr}
	writer := NewResponseWriter(7, output, protocol.TransferTypeStream)
	writer.hasError = true

	if err := writer.Close(); !goerrors.Is(err, wantErr) {
		t.Fatalf("Writer.Close() error = %v, want %v", err, wantErr)
	}
	if output.meta.SessionID != 7 || output.meta.SessionStatus != SessionStatusEnd {
		t.Fatalf("terminal metadata = %+v, want session 7 End", output.meta)
	}
	if !output.meta.Option.Has(OptionError) {
		t.Fatal("terminal metadata lost OptionError")
	}
}
