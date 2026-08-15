// SPDX-License-Identifier: MPL-2.0

package dispatcher

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/pipe"
)

const tcpFlowHistoryLimit = 256

type trackedTCPFlow struct {
	snapshot routing.TCPFlowSnapshot
	uplink   atomic.Int64
	downlink atomic.Int64
}

func (f *trackedTCPFlow) current() routing.TCPFlowSnapshot {
	snapshot := f.snapshot
	snapshot.UplinkBytes = f.uplink.Load()
	snapshot.DownlinkBytes = f.downlink.Load()
	return snapshot
}

type tcpFlowTracker struct {
	mu           sync.RWMutex
	enabled      atomic.Bool
	nextID       atomic.Uint64
	active       map[uint64]*trackedTCPFlow
	closed       []routing.TCPFlowSnapshot
	historyLimit int
	now          func() time.Time
}

func (t *tcpFlowTracker) enable() {
	t.enabled.Store(true)
}

func (t *tcpFlowTracker) isEnabled() bool {
	return t.enabled.Load()
}

func newTCPFlowTracker(historyLimit int) *tcpFlowTracker {
	return &tcpFlowTracker{
		active:       make(map[uint64]*trackedTCPFlow),
		historyLimit: historyLimit,
		now:          time.Now,
	}
}

func (t *tcpFlowTracker) track(link *transport.Link, source, destination, outboundTag string) (*transport.Link, func()) {
	id := t.nextID.Add(1)
	flow := &trackedTCPFlow{snapshot: routing.TCPFlowSnapshot{
		FlowID:      id,
		Source:      source,
		Destination: destination,
		OutboundTag: outboundTag,
		State:       routing.TCPFlowActive,
		StartedAt:   t.now().UTC(),
	}}

	t.mu.Lock()
	t.active[id] = flow
	t.mu.Unlock()

	trackedLink := &transport.Link{Reader: link.Reader, Writer: link.Writer}
	detach := make([]func(), 0, 2)
	if reader, ok := link.Reader.(*pipe.Reader); ok {
		reader.SetReadCounter(func(size int64) { flow.uplink.Add(size) })
		detach = append(detach, func() { reader.SetReadCounter(nil) })
	} else {
		countingReader := &tcpFlowReader{Reader: link.Reader, bytes: &flow.uplink}
		trackedLink.Reader = countingReader
		if timeoutReader, ok := link.Reader.(buf.TimeoutReader); ok {
			trackedLink.Reader = &tcpFlowTimeoutReader{
				tcpFlowReader: countingReader,
				timeoutReader: timeoutReader,
			}
		}
	}
	if writer, ok := link.Writer.(*pipe.Writer); ok {
		writer.SetWriteCounter(func(size int64) { flow.downlink.Add(size) })
		detach = append(detach, func() { writer.SetWriteCounter(nil) })
	} else {
		trackedLink.Writer = &tcpFlowWriter{Writer: link.Writer, bytes: &flow.downlink}
	}
	var closeOnce sync.Once
	return trackedLink, func() {
		closeOnce.Do(func() {
			for _, stop := range detach {
				stop()
			}
			t.close(id)
		})
	}
}

func (t *tcpFlowTracker) close(id uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	flow, found := t.active[id]
	if !found {
		return
	}
	delete(t.active, id)
	snapshot := flow.current()
	snapshot.State = routing.TCPFlowClosed
	snapshot.ClosedAt = t.now().UTC()
	if t.historyLimit == 0 {
		return
	}
	if len(t.closed) == t.historyLimit {
		copy(t.closed, t.closed[1:])
		t.closed[len(t.closed)-1] = snapshot
		return
	}
	t.closed = append(t.closed, snapshot)
}

func (t *tcpFlowTracker) snapshot() []routing.TCPFlowSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snapshots := make([]routing.TCPFlowSnapshot, 0, len(t.active)+len(t.closed))
	snapshots = append(snapshots, t.closed...)
	for _, flow := range t.active {
		snapshots = append(snapshots, flow.current())
	}
	return snapshots
}

type tcpFlowReader struct {
	buf.Reader
	bytes *atomic.Int64
}

type tcpFlowTimeoutReader struct {
	*tcpFlowReader
	timeoutReader buf.TimeoutReader
}

func (r *tcpFlowTimeoutReader) ReadMultiBufferTimeout(timeout time.Duration) (buf.MultiBuffer, error) {
	mb, err := r.timeoutReader.ReadMultiBufferTimeout(timeout)
	r.bytes.Add(int64(mb.Len()))
	return mb, err
}

func (r *tcpFlowReader) ReadMultiBuffer() (buf.MultiBuffer, error) {
	mb, err := r.Reader.ReadMultiBuffer()
	r.bytes.Add(int64(mb.Len()))
	return mb, err
}

func (r *tcpFlowReader) Interrupt() {
	common.Interrupt(r.Reader)
}

type tcpFlowWriter struct {
	buf.Writer
	bytes *atomic.Int64
}

func (w *tcpFlowWriter) WriteMultiBuffer(mb buf.MultiBuffer) error {
	size := int64(mb.Len())
	if err := w.Writer.WriteMultiBuffer(mb); err != nil {
		return err
	}
	w.bytes.Add(size)
	return nil
}

func (w *tcpFlowWriter) Close() error {
	return common.Close(w.Writer)
}

func (w *tcpFlowWriter) Interrupt() {
	common.Interrupt(w.Writer)
}
