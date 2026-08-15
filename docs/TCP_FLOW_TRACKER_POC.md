# TCP flow tracker PoC

## Objective

Prove that the Xray dispatcher can expose one logical TCP flow after its final
outbound handler is selected, without using operating-system socket-table
inference or copying another client's implementation.

## Implemented contract

`routing.TCPFlowInspector` returns detached snapshots with:

- a non-zero ID unique within one dispatcher instance;
- inbound source when available;
- routed destination;
- the actual selected handler tag, including the default handler;
- logical uplink bytes read from the outbound link;
- logical downlink bytes offered to the inbound link;
- `active` and `closed` lifecycle timestamps;
- a closed-flow reason of `completed`, `cancelled`, or `failed`.

Tracking is off by default. An in-process consumer must explicitly call
`EnableTCPFlowTracking()` on the dispatcher inspector before new TCP flows are
recorded. This avoids collecting endpoint metadata or adding pipe callbacks in
an ordinary upstream-compatible run.

Tracking starts only after `routedDispatch` resolves a valid handler. For native
Xray pipes, callbacks are installed without replacing the concrete reader or
writer types used by mux and timeout-aware outbound paths. It ends exactly once
when that handler's synchronous `Dispatch` call returns, or after a submitted
asynchronous mux lifecycle result completes:

- `failed` when the handler reports an error through upstream
  `session.SubmitOutboundErrorToOriginator`; this takes precedence if the
  context is also cancelled;
- `failed` when an active logical mux session receives remote `OptionError`, a
  local mux copy/delivery failure, a terminal-frame write failure, an invalid or
  truncated remote End payload, or its physical worker fails before the logical
  session ends;
- `cancelled` when no failure was reported and `ctx.Err()` is non-nil, or when
  an explicit administrative `ClientWorker.Close()` interrupts the session;
- `completed` otherwise.

The tracker forwards every reported error to the feedback consumer that was
already present in the parent context. Final counters are captured after the
native-pipe counter callbacks are detached; detachment waits for callbacks that
already accepted their bytes, so the closed snapshot cannot freeze before an
in-flight counter update. The tracker retains at most 256 closed snapshots per
dispatcher instance; active flows are not evicted.

For upstream mux with a native pipe reader, `ClientWorker.Dispatch` returns
after assigning the link to a logical mux session. When tracking is enabled,
that session now publishes an internal buffered result channel after cleanup:
`nil` for normal completion or a generic internal error for failure. The
dispatcher flow remains active until the result arrives, so its final counters
include traffic after the asynchronous handoff. The shared physical mux worker
is not assigned an MXR flow ID and may remain alive after the logical flow
closes. No lifecycle channel is allocated when no compatible tracker is present.

## Evidence

Run from the repository root:

```text
go test ./app/dispatcher
go test -race ./app/dispatcher
```

The dispatcher tests cover ordinary completion, an error reported through the
upstream feedback path, and cooperative context cancellation. They verify the
final logical byte counters, one closed snapshot, no remaining active snapshot,
error forwarding, and failure precedence over simultaneous cancellation. A
tracker-level test invokes its finish function twice and verifies that only the
first reason is retained. Separate tests verify that UDP does not create a TCP
flow record and that tracking remains opt-in.

The detour tests cover default fallback after no matching route, a router-picked
handler, and forced-tag precedence over a router result. They verify that the
snapshot and session both retain the actual final outbound tag. Missing routed
and forced handlers create no flow snapshot and leave both sides of the rejected
link closed, matching the existing fail-closed dispatcher path.

The mux tests use the official `ClientWorker` native-pipe path. They verify that
the logical session lifecycle remains open after handoff, returns normal
completion only after an ordinary remote end, returns failure for remote
`OptionError` and physical EOF, returns cancellation for explicit worker close,
returns failure when writing a terminal End frame fails, strictly validates data
attached to a remote End, and closes every affected active session without
closing the physical worker after an ordinary logical end. A deterministic
reservation test proves that a concurrent local EOF cannot publish `completed`
while a remote End payload is still being validated. A concurrent Dispatch/Close
test covers the allocation-to-link-binding boundary. A stalled remote End test
proves that request cancellation still closes the logical session, while remote
`OptionError` retains failure precedence. Pipe tests prove that callback detach
waits for already in-flight read and write accounting. Dispatcher tests
independently verify that the tracker waits for the asynchronous result before
freezing counters, forwards a lifecycle failure once, does not forward
cancellation as an error, and records the corresponding terminal reason.

## Current validation record

On 2026-08-15 with Go 1.26.0 on Windows:

- `go test ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher` — passed;
- `go test -race ./common/mux ./transport/pipe ./app/dispatcher` — passed;
- targeted MUX physical EOF, administrative close, terminal write failure,
  remote End payload, and concurrent Dispatch/Close race tests repeated 100
  times — passed;
- targeted dispatcher lifecycle failure/cancellation race tests repeated 100
  times — passed;
- `go test -run '^$' -bench '^BenchmarkRoutedDispatchTCPFlowTracking$' -benchmem -benchtime=500ms -count=5 '-cpu=1,8' ./app/dispatcher` — passed;
- `go test -run '^$' -bench '^BenchmarkTCPFlowPipeByteAccounting$' -benchmem -benchtime=1s -count=5 '-cpu=2' ./app/dispatcher` — passed;
- `go test ./...` — not green on the clean upstream baseline environment:
  DNS integration expected an external response, and geodata tests could not
  find `resources/geoip.dat` and `resources/geosite.dat`; the remaining hung
  integration process was stopped after those failures. These failures did not
  occur in the changed target packages.

## Benchmark result and PoC decision

On the recorded Windows host, median native-pipe lifecycle results were:

- tracker absent: 50.43 ns/op, 24 B/op, 1 alloc/op on one CPU;
- tracker present but disabled: 51.00 ns/op, 24 B/op, 1 alloc/op on one CPU;
- tracker enabled: 3.792 microseconds/op, 640 B/op, 19 allocs/op on one CPU;
- tracker enabled with `-cpu=8`: 1.001 microseconds/op aggregate, with the same
  bytes and allocations per flow.

The enabled incremental lifecycle cost over the absent baseline is therefore
about 3.742 microseconds, 616 bytes, and 18 allocations per synthetic flow while
the 256-record history is saturated and concurrent writers contend on current
locks. For 1 KiB native-pipe traffic, median uplink reads were 101.6 ns/op with
accounting disabled and 106.1 ns/op enabled; median downlink writes were 100.7
ns/op disabled and 113.5 ns/op enabled. The exact-final-counter barrier therefore
adds measurable CPU cost in this microbenchmark, while both directions retain 2
allocations per operation.

Decision: **KEEP** the tracker as an opt-in, in-process experiment. The disabled
path is indistinguishable from the no-tracker baseline in this benchmark, the
per-flow enabled cost is bounded in absolute terms, and steady-state byte
accounting adds no allocation count but has measurable CPU overhead. This
decision does not promote the snapshot shape to a stable API. The synthetic
benchmark reports aggregate throughput rather than tail latency and does not
cover future external core API delivery, live-network behavior, or published
core artifacts.

## Known limits

- Counters describe logical payload at the dispatcher link, not IP/TCP headers
  or physical-interface bytes.
- A write is counted when accepted by the pipe/writer; it is not an
  acknowledgement of remote receipt.
- The FlowID remains a dispatcher-flow ID. The PoC does not expose the mux
  protocol SessionID or assign an ID to the shared physical mux connection.
- An abnormal physical mux-worker termination before a logical end marks every
  still-active logical session `failed`; an explicit administrative worker close
  marks them `cancelled`. The first worker-close source wins. Abnormal termination
  intentionally uses a generic internal error rather than preserving a low-level
  transport or process cause.
- Mux terminal attribution is first-close-wins. Once an active tracked session
  observes a normal remote End, it suppresses only a concurrent normal uplink
  EOF until attached data is fully validated, so that EOF cannot publish a false
  `completed`. An already-known remote `OptionError` closes as `failed`
  immediately; cancellation or another non-nil terminal result may still win
  while validation stalls. A remote End cannot revise a session that had already
  completed before the End was observed.
- An error before the first mux payload is treated as failure, matching the
  upstream writer's `OptionError` behavior, although a zero-payload TCP session
  remains an ambiguous edge case.
- A handler that fails without using the upstream error-feedback path is
  indistinguishable from normal completion because `outbound.Handler.Dispatch`
  has no error return.
- Context cancellation does not forcibly interrupt a handler. A handler that
  ignores cancellation remains active until its synchronous `Dispatch` call
  returns, preserving truthful final counters instead of closing early.
- Cancelling a tracked logical session closes that flow even when a remote End
  payload stalls, but it does not tear down the shared physical mux worker. The
  worker's frame parser can remain stalled until the peer resumes or the worker
  is closed; timeout/recovery policy is outside this PoC.
- There is no protobuf/gRPC/REST API, subscription stream, persistence, redaction
  policy, or production retention configuration.
- No versioned external core API or published core artifact currently consumes
  this inspector.
- Snapshot polling contention and tail latency are not benchmarked because no
  external snapshot/event delivery contract exists yet.

## Promotion gates

Before this becomes a stable MXR Core contract, continue in this order:

1. a versioned external core snapshots/events API with bounded delivery,
   retention, redaction, and authorization;
2. a narrow versioned core embedding boundary and reproducible core
   library/binary artifacts;
3. core threat modeling, MPL notices, provenance, SBOM, signing, and release
   validation.
