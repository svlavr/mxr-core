# AGENTS.md

## Project identity

- This repository is the independent MXR Core fork of official
  `XTLS/Xray-core`.
- Fetch base source only from `https://github.com/XTLS/Xray-core.git` and record
  the exact commit in `docs/FORK_BASELINE.md`. Never substitute an older local
  checkout for an upstream update.
- Keep the upstream module path unless an accepted build or publication gate
  proves that changing it is necessary.
- The inherited source remains under MPL-2.0. Do not remove upstream copyright,
  license, security, or contribution files.

## Repository boundaries

- `upstream` is read-only evidence and the source of synchronization. Never push
  fork branches to it.
- The MXR Core GitHub remote must be named `origin`. Creating it, choosing its
  visibility, and publishing branches require explicit user authorization.
- External clients and GPL reference trees are evidence only. Do
  not import their source, resources, modules, or generated artifacts.
- Keep credentials, live endpoints, captures, device evidence, signing material,
  and private configurations under ignored `.local/` paths.

## Development rules

- The active queue is `docs/ROADMAP.md`; do not turn future items into code while
  implementing a bounded slice.
- Prefer small, upstream-mergeable patches. Do not rewrite unrelated upstream
  code or regenerate broad protobuf surfaces for a local experiment.
- A missing behavior is an explicit gap. Do not hide it with a stub, synthetic
  production result, or silent compatibility fallback.
- Before editing, inspect the current branch, status, baseline, and affected
  upstream call path. Preserve unrelated work.
- After changes, run the narrowest affected tests and `go test -race` when the
  changed code has shared mutable state. Report source/build, Android/device,
  live-network, and release evidence separately.
- Treat the TCP flow tracker as experimental until its contract, overhead,
  lifecycle semantics, public API, and Android integration gates are accepted.

## Current fork extension

- The first accepted experiment is a TCP flow tracker at the final dispatcher
  selection boundary.
- It may expose a dispatcher-instance flow ID, source, destination, final
  outbound tag, logical uplink/downlink byte counts, and active/closed lifecycle.
- Current non-goals are UDP flow identity, mux substream identity, API transport,
  persistence, Android `VpnService`, UI, billing, and release readiness.

## Local skills

- Use `.agents/skills/xray-fork-research` before a new upstream divergence,
  runtime mechanism, external dependency, or licensing decision.
- Use `.agents/skills/xray-fork-implement` for approved code and documentation.
- Use `.agents/skills/xray-fork-android-validation` only when an executable
  Android path exists or the user requests Android validation.
- Use `.agents/skills/adversarial-reviewer` after a substantial runtime change.
- Use `.agents/skills/xray-fork-checkpoint` only within its remote and safety
  gates; it must never create a remote or change repository visibility.
