// SPDX-License-Identifier: MPL-2.0

package dispatcher

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/buf"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/session"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/transport"
	"github.com/xtls/xray-core/transport/pipe"
)

func BenchmarkRoutedDispatchTCPFlowTracking(b *testing.B) {
	for _, test := range []struct {
		name        string
		withTracker bool
		enabled     bool
	}{
		{name: "absent"},
		{name: "disabled", withTracker: true},
		{name: "enabled_native_pipe", withTracker: true, enabled: true},
	} {
		b.Run(test.name, func(b *testing.B) {
			handler := new(flowBenchmarkHandler)
			dispatcher := &DefaultDispatcher{
				ohm: &flowTestOutboundManager{handler: handler},
			}
			if test.withTracker {
				dispatcher.tcpFlows = newTCPFlowTracker(tcpFlowHistoryLimit)
			}
			if test.enabled {
				dispatcher.EnableTCPFlowTracking()
			}

			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				destination := net.TCPDestination(net.DomainAddress("example.com"), 443)
				ctx := session.ContextWithOutbounds(context.Background(), []*session.Outbound{{Target: destination}})
				uplinkReader, uplinkWriter := pipe.New(pipe.WithoutSizeLimit())
				downlinkReader, downlinkWriter := pipe.New(pipe.WithoutSizeLimit())
				defer common.Close(uplinkWriter)
				defer common.Close(downlinkReader)
				link := &transport.Link{Reader: uplinkReader, Writer: downlinkWriter}
				for pb.Next() {
					dispatcher.routedDispatch(ctx, link, destination)
				}
			})
		})
	}
}

func BenchmarkTCPFlowPipeByteAccounting(b *testing.B) {
	const payloadSize = 1024
	for _, direction := range []struct {
		name     string
		downlink bool
	}{
		{name: "uplink_read"},
		{name: "downlink_write", downlink: true},
	} {
		for _, test := range []struct {
			name    string
			enabled bool
		}{
			{name: "disabled"},
			{name: "enabled", enabled: true},
		} {
			b.Run(direction.name+"/"+test.name, func(b *testing.B) {
				uplinkReader, uplinkWriter := pipe.New(pipe.WithoutSizeLimit())
				downlinkReader, downlinkWriter := pipe.New(pipe.WithoutSizeLimit())
				tracker := newTCPFlowTracker(tcpFlowHistoryLimit)
				link := &transport.Link{Reader: uplinkReader, Writer: downlinkWriter}
				var finish func(routing.TCPFlowEndReason)
				if test.enabled {
					link, finish = tracker.track(link, "", "tcp:example.com:443", "benchmark-out")
				}
				var writeTarget buf.Writer = uplinkWriter
				var readSource buf.Reader = link.Reader
				if direction.downlink {
					writeTarget = link.Writer
					readSource = downlinkReader
				}

				payload := make([]byte, payloadSize)
				writeErr := make(chan error, 1)
				b.SetBytes(payloadSize)
				b.ReportAllocs()
				b.ResetTimer()
				go func() {
					for range b.N {
						if err := writeTarget.WriteMultiBuffer(buf.MultiBuffer{buf.FromBytes(payload)}); err != nil {
							writeErr <- err
							return
						}
					}
					writeErr <- common.Close(writeTarget)
				}()

				var total int64
				want := int64(b.N * payloadSize)
				for total < want {
					mb, err := readSource.ReadMultiBuffer()
					if err != nil {
						b.Fatal(err)
					}
					total += int64(mb.Len())
					mb = buf.ReleaseMulti(mb)
				}
				if err := <-writeErr; err != nil {
					b.Fatal(err)
				}
				b.StopTimer()

				if test.enabled {
					finish(routing.TCPFlowCompleted)
					flows := tracker.snapshot()
					if len(flows) != 1 {
						b.Fatalf("tracked flows = %v, want one", flows)
					}
					got := flows[0].UplinkBytes
					if direction.downlink {
						got = flows[0].DownlinkBytes
					}
					if got != want {
						b.Fatalf("tracked bytes = %d, want %d", got, want)
					}
				}
				common.Close(uplinkWriter)
				common.Close(downlinkReader)
			})
		}
	}
}

type flowBenchmarkHandler struct{}

func (*flowBenchmarkHandler) Tag() string                               { return "benchmark-out" }
func (*flowBenchmarkHandler) Dispatch(context.Context, *transport.Link) {}
func (*flowBenchmarkHandler) SenderSettings() *serial.TypedMessage      { return nil }
func (*flowBenchmarkHandler) ProxySettings() *serial.TypedMessage       { return nil }
func (*flowBenchmarkHandler) Start() error                              { return nil }
func (*flowBenchmarkHandler) Close() error                              { return nil }
