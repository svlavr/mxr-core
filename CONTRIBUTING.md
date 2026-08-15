# Contributing to MXR Core

## Before opening a change

1. Search existing issues and pull requests.
2. Describe the exact runtime gap and whether official upstream already solves it.
3. Keep the patch small and avoid unrelated formatting or generated-file churn.
4. Do not include credentials, live endpoints, captures, private configuration,
   signing material, binaries, or external client source.

## Required checks

Run the checks relevant to the changed path. For the current flow tracker:

```text
go test ./transport/pipe ./features/routing ./app/dispatcher
go test -race ./app/dispatcher
go vet ./transport/pipe ./features/routing ./app/dispatcher
```

Add tests for lifecycle, error, cancellation, concurrency, and ownership changes.
Report host, Android build, emulator, physical-device, live-network, and release
evidence separately.

## Pull requests

- Explain what changed, why it belongs in the fork, and what remains unproven.
- Link an issue for material runtime or public-contract changes.
- Do not weaken branch protection, CI permissions, or secret boundaries.
- Contributions are provided under MPL-2.0.
