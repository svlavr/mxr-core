package pipe

import (
	"github.com/xtls/xray-core/common/buf"
)

// Writer is a buf.Writer that writes data into a pipe.
type Writer struct {
	pipe *pipe
}

// WriteMultiBuffer implements buf.Writer.
func (w *Writer) WriteMultiBuffer(mb buf.MultiBuffer) error {
	return w.pipe.WriteMultiBuffer(mb)
}

// Close implements io.Closer. After the pipe is closed, writing to the pipe will return io.ErrClosedPipe, while reading will return io.EOF.
func (w *Writer) Close() error {
	return w.pipe.Close()
}

func (w *Writer) Len() int32 {
	return w.pipe.Len()
}

// Interrupt implements common.Interruptible.
func (w *Writer) Interrupt() {
	w.pipe.Interrupt()
}

// SetWriteCounter sets an optional callback invoked for bytes accepted by this
// pipe and returns an idempotent detach function for that callback generation.
// Detach waits for its invocations already in flight and must not be called by
// the callback itself.
func (w *Writer) SetWriteCounter(counter func(int64)) func() {
	return w.pipe.setWriteCounter(counter)
}
