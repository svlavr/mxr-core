# Fork roadmap

This is the only active queue for MXR Core. Completed evidence belongs in the
owning technical document; external implementations belong in the donor
register and do not become roadmap items automatically.

## Verified foundation

- [x] Clone the official upstream `main` and record the exact baseline commit.
- [x] Implement the opt-in TCP flow tracker after final outbound selection.
- [x] Cover completion, failure, cancellation, detours, logical MUX lifecycle,
  exact final counters, concurrency, and retained-history benchmarks.
- [x] Keep the tracker disabled by default and accept it as an internal `KEEP`
  experiment, not a stable public contract.
- [x] Publish the public repository with MPL-2.0 notices, protected `main`, and
  required Linux, race, and Windows checks.

The implemented behavior, measurements, and remaining limitations are recorded
in `TCP_FLOW_TRACKER_POC.md`.

## Active next gate: external inspection contract

Do not add transport or binding code until this contract is accepted. Define:

- a versioned core-owned snapshot/event model and its compatibility rule;
- ordering through sequence numbers, gap reporting, and snapshot watermarks;
- bounded retention, queue limits, overflow behavior, and reconnection;
- authorization plus source/destination redaction before data leaves the core;
- clear/reset semantics and identifier lifetime across core restarts;
- benchmark gates for polling, delivery contention, allocations, and tail
  latency.

Acceptance requires an API proposal, concurrency tests, privacy boundaries, and
measured overhead. The existing in-process inspector remains unchanged until
those items are reviewed together.

## Later gates

1. Define the smallest versioned core embedding API and reproducible
   library/binary artifacts, including build provenance and shutdown tests.
2. Make broader upstream integration validation deterministic by controlling
   geodata and live-network dependencies.
3. Complete the core threat model, MPL corresponding-source review, SBOM,
   signing, reproducible release, rollback, and release acceptance gates.

This repository is limited to core runtime behavior, core APIs, and core
artifacts.
