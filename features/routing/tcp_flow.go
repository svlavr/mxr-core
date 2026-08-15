// SPDX-License-Identifier: MPL-2.0

package routing

import "time"

// TCPFlowState describes the lifecycle state of a tracked TCP flow.
type TCPFlowState string

const (
	TCPFlowActive TCPFlowState = "active"
	TCPFlowClosed TCPFlowState = "closed"
)

// TCPFlowSnapshot is a point-in-time view of one routed TCP flow.
// FlowID is unique within one dispatcher instance.
type TCPFlowSnapshot struct {
	FlowID        uint64
	Source        string
	Destination   string
	OutboundTag   string
	UplinkBytes   int64
	DownlinkBytes int64
	State         TCPFlowState
	StartedAt     time.Time
	ClosedAt      time.Time
}

// TCPFlowInspector exposes active and recently closed TCP flows.
// Implementations must return detached snapshots safe for concurrent readers.
type TCPFlowInspector interface {
	EnableTCPFlowTracking()
	SnapshotTCPFlows() []TCPFlowSnapshot
}
