# Donor patch register

## Purpose

This register records external Xray-core implementations that may reduce
fork-specific work without making another fork authoritative for MXR Core. An
entry is evidence, not approval to import a branch, API, module, or policy.

Every adopted change must remain a small reviewed patch against the exact
baseline in `FORK_BASELINE.md`. Prefer official upstream behavior whenever it
already provides the required core mechanism.

Evidence was reviewed on 2026-08-15. Recheck mutable PR heads before any rebase
experiment.

## Scope and status

Only Xray-core source changes, core APIs, core artifacts, and regression gates
belong here. Everything outside that boundary is not a donor candidate for this
repository.

- `UPSTREAM_NATIVE` — use the official baseline capability; no donor patch.
- `PATCH_CANDIDATE` — selected for a bounded experiment, not accepted.
- `PATTERN_ONLY` — reuse only the contract or test idea after independent design.
- `DEFERRED` — relevant to a later roadmap gate and not authorized now.
- `REJECTED` — inspected and deliberately excluded.

## Candidates

### Connection registry and forced close

- Evidence: [XTLS/Xray-core PR #5844](https://github.com/XTLS/Xray-core/pull/5844),
  open at `f356a34cf8f9fe352647934248f993938e29b85c`, 34 changed files.
- Useful subset: instance-owned manager shape, TCP/UDP wrapping locations,
  cancellation mechanics, registry behavior, and race scenarios.
- Status: `PATCH_CANDIDATE`.
- Decision: do not import its command/API surface or whole patch. It has no MXR
  final-route receipt or generation identity. Its 64-entry subscriber channels
  silently drop events for a slow consumer and cannot define the external event
  contract. Merged [PR #5732](https://github.com/XTLS/Xray-core/pull/5732)
  supplies user/IP online-map accounting, not a per-flow registry.

### Passive terminal flow failure

- Evidence: baseline `common/session.TrackedConnectionError` and
  `SubmitOutboundErrorToOriginator`, already used by DNS, MUX, and outbound
  transport failure paths.
- Useful subset: bridge the existing raw terminal signal to a tracked flow after
  final outbound selection.
- Status: `UPSTREAM_NATIVE`.
- Decision: preserve the original observer and report only a raw flow outcome.
  A passive failure is not node-health, scoring, or failover authority.

### Exact tagged TCP/UDP probe transport

- Evidence: baseline `tagged.Dialer`, `core.Dial`, `core.DialUDP`, and the
  existing `github.com/pion/stun/v3` dependency.
- Useful subset: forced-outbound TCP/UDP dispatch and bounded raw STUN
  measurements for RTT, loss variation, and mapped egress address.
- Status: `UPSTREAM_NATIVE`.
- Decision: no wrapper donor or new packet engine is needed. A later fork patch
  may add a narrow typed raw-result boundary, but not selection or failover
  policy.

### Bounded one-shot probe batches

- Evidence: [XTLS/Xray-core PR #6492](https://github.com/XTLS/Xray-core/pull/6492),
  closed without merge at `1f0baf4cd19b4712f9c56e5e2d9ccd812efee1dd`,
  based on `50231eaff98ccc31b5cbd247a721c16e97fe5ec1`.
- Useful subset: worker limits, repeated samples, deadlines, cancellation,
  partial results, progressive notifications, and tests.
- Status: `DEFERRED`.
- Decision: rebase and rerun the tests only if this capability enters the active
  roadmap. The PR does not define generation identity, traffic budgets, or a
  typed external receipt.

### Protocol-safe forced close

- Evidence: [XTLS/Xray-core PR #6504](https://github.com/XTLS/Xray-core/pull/6504),
  closed without merge at `fc3777a76b9a56a1867226210b021b338b067871`,
  one changed file based on `50231eaff98ccc31b5cbd247a721c16e97fe5ec1`.
- Useful subset: `Close` forwarding for Vision reader/writer wrappers.
- Status: `PATCH_CANDIDATE`.
- Decision: require a reproducing forced-close test plus idempotent and
  concurrent-close review before adoption. The donor contains no regression
  test.

### Leak-free tagged dial close

- Evidence: [XTLS/Xray-core PR #6583](https://github.com/XTLS/Xray-core/pull/6583),
  closed without merge at `c0f42551bffb8a1a22f3c0ceeb5e945632ecb723`,
  one changed file based on `5ca6f4b7d4dc20a881d4330e498892697627ec0c`.
- Useful subset: cancel a dispatched outbound when the returned tagged
  connection closes, including a dial or handshake stalled before the copy loop.
- Status: `PATCH_CANDIDATE`.
- Decision: require a stalled-peer regression test, exactly-once cleanup, and
  TCP/UDP behavior review. The donor contains no regression test.

### Multiple Xray instances

- Evidence: [XTLS/Xray-core PR #6483](https://github.com/XTLS/Xray-core/pull/6483),
  closed without merge at `b4f634162bce9c89a8f2232f7f8d706d022a4d41`,
  based on `50231eaff98ccc31b5cbd247a721c16e97fe5ec1`.
- Useful subset: two commits attempt to scope system-dialer dependencies to an
  instance; the branch also contains an unrelated Observatory commit.
- Status: `REJECTED` as a whole.
- Decision: simultaneous live core instances are not part of the accepted
  contract. Revisit only after an explicit core architecture decision.

### Race-safe ownership transfer

- Evidence: [Jolymmiles commit `e7b23ab`](https://github.com/Jolymmiles/Xray-core/commit/e7b23ab1059db51bdba23035fcabc0a6af8ff10b).
- Useful subset: one-shot `Prepare` to `Activate` or `Abort`, lease ownership,
  atomic handoff, instance generation, and race tests.
- Status: `PATTERN_ONLY`.
- Decision: the donor models authenticated online-presence references, not a
  general connection lifecycle. Do not copy its identity or online-map model.

### Capability and build manifest

- Evidence: baseline feature registration, handler managers, forced outbound
  selection, stats, and protobuf configuration.
- Useful subset: typed receipts and reproducible build provenance around
  existing upstream mechanisms.
- Status: `UPSTREAM_NATIVE` for core mechanics.
- Decision: no broad donor patch is selected. Transactional validation and
  release provenance remain fork work.

## Adoption gates

Before copying or adapting donor lines:

1. Freeze the exact repository, commit, file set, and retrieval date.
2. Compare the donor base with the current `FORK_BASELINE.md` commit.
3. Extract only required mechanics; reject unrelated policy, command/API
   surfaces, generated files, and refactors.
4. Confirm MPL-2.0 compatibility, retain notices and attribution, and record
   modified covered files.
5. Use an explicit reviewed patch rather than merging a donor branch.
6. Run focused tests, race tests for shared state, and hot-path benchmarks.
7. Review cancellation, ownership, event loss, identifier reuse, instance
   isolation, and resource cleanup adversarially.
8. Record the accepted fork commit or the rejection here.

An external implementation estimate is not acceptance evidence. Schedule and
effort savings must come from a rebased diff and passing gates.
