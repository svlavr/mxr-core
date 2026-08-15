# TCP flow tracker PoC

## Objective

Prove that the Xray dispatcher can expose one logical TCP flow after its final
outbound handler is selected, without using Android socket-table inference or
copying another client's implementation.

## Implemented contract

`routing.TCPFlowInspector` returns detached snapshots with:

- a non-zero ID unique within one dispatcher instance;
- inbound source when available;
- routed destination;
- the actual selected handler tag, including the default handler;
- logical uplink bytes read from the outbound link;
- logical downlink bytes offered to the inbound link;
- `active` and `closed` lifecycle timestamps.

Tracking is off by default. An in-process consumer must explicitly call
`EnableTCPFlowTracking()` on the dispatcher inspector before new TCP flows are
recorded. This avoids collecting endpoint metadata or adding pipe callbacks in
an ordinary upstream-compatible run.

Tracking starts only after `routedDispatch` resolves a valid handler. For native
Xray pipes, callbacks are installed without replacing the concrete reader or
writer types used by mux and timeout-aware outbound paths. It ends when that
handler's synchronous `Dispatch` call returns. The tracker retains at most 256
closed snapshots per dispatcher instance; active flows are not evicted.

## Evidence

Run from the repository root:

```text
go test ./app/dispatcher
go test -race ./app/dispatcher
```

The integration test drives `DefaultDispatcher.routedDispatch` through a fake
outbound handler, observes the active snapshot while the handler is blocked,
then verifies final counters and the closed snapshot. A separate test verifies
that UDP does not create a TCP flow record, and another verifies opt-in behavior.

## Current validation record

On 2026-08-15 with Go 1.26.0 on Windows:

- `go test ./transport/pipe ./features/routing ./app/dispatcher` — passed;
- `go test -race ./app/dispatcher` — passed;
- `go test ./...` — not green on the clean upstream baseline environment:
  DNS integration expected an external response, and geodata tests could not
  find `resources/geoip.dat` and `resources/geosite.dat`; the remaining hung
  integration process was stopped after those failures. These failures did not
  occur in the changed target packages.

## Known limits

- Counters describe logical payload at the dispatcher link, not IP/TCP headers
  or physical-interface bytes.
- A write is counted when accepted by the pipe/writer; it is not an
  acknowledgement of remote receipt.
- Mux substreams are not assigned independent IDs by this PoC.
- There is no protobuf/gRPC/REST API, subscription stream, persistence, redaction
  policy, or production retention configuration.
- No Android artifact or application currently consumes this inspector.
- Performance and allocation overhead have not yet been benchmarked.

## Promotion gates

Before this becomes a stable MXR Core contract, decide and test:

1. lifecycle semantics for mux, detours, cancellation, and handler failures;
2. bounded event delivery and slow-consumer behavior;
3. identifiers across core restarts and multiple core instances;
4. sensitive-address redaction and API authorization;
5. benchmarks under concurrent TCP load;
6. Android integration and physical-device behavior.
