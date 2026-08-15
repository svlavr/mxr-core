# Project setup gaps

The fork is locally usable for the TCP flow tracker experiment, but it is not a
published, integrated, Android-ready, or release-ready product.

## Repository and provenance

- The public GitHub repository is `https://github.com/svlavr/mxr-core` and is
  configured as `origin`.
- Publish and protect `main`; pushes to official `upstream` remain disabled.
- Keep `main` as the permanent default branch and synchronize from the exact
  official upstream SHA through reviewed branches.
- Fork ownership (`CODEOWNERS`), contribution policy, and PR templates are
  present. A fork changelog and release provenance remain open.
- Define how exact upstream SHAs, applied fork commits, Go toolchain, modules,
  and generated artifacts are reproduced and audited.

## CI and test assets

- Windows and Linux CI covers formatting, focused tests, race tests, and vet.
  A reproducible release build remains open.
- Make the broader upstream test gate deterministic: provide controlled geodata
  assets and isolate tests that currently depend on live DNS/network responses.
- Add benchmarks for tracker CPU, allocations, lock contention, and retained
  history under concurrent flows.
- Add lifecycle cases for route failure, forced/default outbound, cancellation,
  failed writes, sniffing/route-only mode, detours, and mux.

## Stable fork contract

- Decide whether the tracker remains in-process only or gains a versioned
  protobuf/API/event contract.
- Define subscription backpressure, overflow, ordering, reconnection, retention,
  clear/reset behavior, and multiple-core identity.
- Define sensitive source/destination redaction and authorization before exposing
  snapshots outside the process.
- Decide exact semantics for logical payload bytes versus transport/interface
  bytes and for mux substreams.

## Product integration

- Design a clean consumer boundary from the app to this repository; do not copy
  external application modules into the fork.
- Add a reproducible Android library/binary build, supported ABIs, API 29
  compatibility, binding/JNI contract, native loading, packaging, and shutdown.
- Only then connect to a target-owned Android `VpnService`/TUN path and validate
  emulator, physical-device, background lifecycle, network changes, and load.

## Security, licensing, and release

- Complete a threat model for configuration, control API, routing/DNS integrity,
  endpoint metadata, native integration, updates, and build publication.
- Add dependency/SBOM, vulnerability, secret, license, and artifact-signing
  checks.
- Before distributing an APK or core binary, complete MPL-2.0 notice and
  corresponding-source review for modified covered files.
- Create privacy disclosures, data-retention behavior, Play policy checks,
  signing/key custody, internal testing track, crash/telemetry policy, rollback,
  and release acceptance gates.
