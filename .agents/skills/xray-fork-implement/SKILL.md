---
name: xray-fork-implement
description: Implement an approved, bounded change in MXR Core while preserving upstream mergeability and honest validation boundaries.
---

# Xray fork implementation

1. Read the root `AGENTS.md`, `docs/FORK_BASELINE.md`, `docs/ROADMAP.md`, and only
   the directly relevant technical document.
2. Confirm the requested outcome, exact write-set, current branch/status, dirty
   overlap, upstream call path, and smallest useful evidence gate.
3. For a new upstream divergence, external dependency, runtime/security contract,
   or license decision, use `xray-fork-research` first.
4. Implement the smallest coherent vertical slice. Prefer fork-owned files and a
   narrow hook over broad upstream rewrites, duplicate protocol models, stubs, or
   speculative factories.
5. Preserve `transport.Link` ownership, cancellation, close/interrupt behavior,
   and concurrency guarantees. Treat dispatcher, proxy, mux, DNS, config parsing,
   and native/platform boundaries as security-sensitive.
6. Run affected Go tests. Use `go test -race` for shared state, concurrency, flow
   lifecycle, and event delivery. Escalate to broader tests only when imports or
   generated contracts require it.
7. Use `xray-fork-android-validation` only when an executable Android boundary is
   affected. Keep host source/build evidence separate from emulator, device,
   live-network, and release evidence.
8. After a substantial runtime change, run `adversarial-reviewer`. Update the
   owning fork document and roadmap with proven status and explicit non-goals.
9. Use `xray-fork-checkpoint` only after the slice is complete and its remote
   gates are satisfied. Never create a remote or change visibility implicitly.
