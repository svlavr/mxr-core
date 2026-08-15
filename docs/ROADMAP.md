# Fork roadmap

This is the only active queue for this repository. Items are ordered; later
items are not implied authorization for implementation.

## Active: TCP flow tracker PoC

- [x] Clone official latest `main` and record exact baseline.
- [x] Track one routed TCP flow after final outbound selection.
- [x] Expose source, destination, final outbound, logical bytes, and close state.
- [x] Add focused dispatcher tests.
- [x] Pass race test and complete adversarial review on current bytes.
- [x] Close exactly once with `completed`, `cancelled`, or `failed` after the
  synchronous handler returns; preserve final counters and upstream error
  feedback for ordinary completion, reported failure, and cooperative context
  cancellation.
- [x] Test default fallback, routed selection, forced-tag precedence, and
  missing routed/forced handlers as a separate detour slice.
- [x] Keep each dispatcher flow active through its logical mux session cleanup
  while leaving the shared physical mux worker outside flow identity/lifecycle.
- [x] Attribute an abnormal physical mux-worker termination to every
  still-active logical session as `failed`, treat explicit administrative
  worker close as `cancelled`, preserve normal logical completion, and honor
  the protocol's remote `OptionError`; the first terminal session event wins.
- [x] Benchmark native-pipe lifecycle CPU, allocations, and lock contention
  under concurrent TCP load, plus steady-state uplink/downlink accounting.
- [x] Keep the opt-in in-process experiment. This is not promotion to a stable
  external API or production-ready contract.

## Next only after PoC acceptance

- [ ] Define a versioned external event/snapshot contract.
- [ ] Specify retention, backpressure, redaction, and authorization.
- [ ] Design the Android consumer boundary without importing external
  application code.

## Publication gate

- [x] Choose GitHub owner and public visibility.
- [x] Create the public `origin` repository.
- [x] Publish the initial `main`.
- [x] Enable and verify branch protection.
- [x] Verify CI, dependency/security settings, notices, and repository permissions.
- [ ] Add release provenance before publishing any binary.
- [ ] Complete Android, Play policy, privacy, signing, and release validation
  before making any Google Play readiness claim.
