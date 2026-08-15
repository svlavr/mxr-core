# Fork roadmap

This is the only active queue for this repository. Items are ordered; later
items are not implied authorization for implementation.

## Active: TCP flow tracker PoC

- [x] Clone official latest `main` and record exact baseline.
- [x] Track one routed TCP flow after final outbound selection.
- [x] Expose source, destination, final outbound, logical bytes, and close state.
- [x] Add focused dispatcher tests.
- [x] Pass race test and complete adversarial review on current bytes.
- [ ] Decide whether to keep, revise, or discard the experiment.

## Next only after PoC acceptance

- [ ] Define a versioned external event/snapshot contract.
- [ ] Specify retention, backpressure, redaction, and authorization.
- [ ] Test detours, failures, cancellation, and mux semantics.
- [ ] Benchmark CPU, allocation, and lock overhead.
- [ ] Design the Android consumer boundary without importing external application code.

## Publication gate

- [x] Choose GitHub owner and public visibility.
- [x] Create the public `origin` repository.
- [x] Publish the initial `main`.
- [ ] Enable and verify branch protection.
- [ ] Verify CI, dependency/security settings, notices, and repository permissions.
- [ ] Add release provenance before publishing any binary.
- [ ] Complete Android, Play policy, privacy, signing, and release validation
  before making any Google Play readiness claim.
