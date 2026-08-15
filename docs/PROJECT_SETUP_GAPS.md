# Project setup gaps

The fork is locally usable for the TCP flow tracker experiment, but it does not
yet expose a stable external core API or a release-ready core artifact.

## Repository and provenance

- The public GitHub repository is `https://github.com/svlavr/mxr-core` and is
  configured as `origin`.
- `main` is published and protected by required pull requests and the `verify`,
  `race`, and `windows` checks. Pushes to official `upstream` remain disabled.
- Keep `main` as the permanent default branch and synchronize from the exact
  official upstream SHA through reviewed branches.
- Fork ownership (`CODEOWNERS`), contribution policy, and PR templates are
  present. A fork changelog and release provenance remain open.
- Define how exact upstream SHAs, applied fork commits, Go toolchain, modules,
  and generated artifacts are reproduced and audited.

## CI and test assets

- Verified Windows and Linux CI covers formatting, focused tests, race tests,
  and vet.
  A reproducible release build remains open.
- Make the broader upstream test gate deterministic: provide controlled geodata
  assets and isolate tests that currently depend on live DNS/network responses.
- Repeat performance gates when snapshot polling or an event queue exists;
  current benchmarks cover saturated retained history, concurrent lifecycle
  writes, the exact-final-counter detach barrier, and native-pipe
  uplink/downlink accounting, but not polling or tail latency.
- Add the remaining lifecycle cases for non-mux failed writes and
  sniffing/route-only mode. Ordinary completion, reported failure, cooperative
  cancellation, default/routed/forced selection, missing-detour rejection,
  logical mux-session cleanup, remote mux error, terminal-frame write failure,
  strict remote End-payload validation, and physical mux-worker termination are
  covered.

## Stable fork contract

- Decide whether the tracker remains in-process only or gains a versioned
  external core API/event contract.
- Define subscription backpressure, overflow, ordering, reconnection, retention,
  and clear/reset behavior.
- Define sensitive source/destination redaction and authorization before exposing
  snapshots outside the process.
- Decide exact semantics for logical payload bytes versus transport/interface
  bytes and for mux substreams.

## Core API and artifacts

- Define the smallest versioned core embedding API needed to configure, start,
  inspect, cancel, and stop the core runtime.
- Add reproducible core library/binary builds, supported target and architecture
  metadata, a narrow exported binding surface, deterministic loading, and
  shutdown tests.
- Keep this repository limited to core APIs, core runtime code, and core
  artifacts.

## Security, licensing, and release

- Complete a threat model for configuration, the external core API, routing/DNS
  integrity, endpoint metadata, updates, and build publication.
- Dependabot security updates, secret scanning, push protection, and private
  vulnerability reporting are enabled. SBOM, license automation, and artifact
  signing remain open.
- Before distributing a core library or binary, complete MPL-2.0 notice and
  corresponding-source review for modified covered files.
- Define core artifact signing/key custody, SBOM, reproducible build, rollback,
  and release acceptance gates.
