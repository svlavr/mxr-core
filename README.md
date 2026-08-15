# MXR Core

MXR Core is an experimental fork of
[XTLS/Xray-core](https://github.com/XTLS/Xray-core). The fork keeps upstream
history and focuses on small, reviewable runtime experiments.

## Current experiment

The first change is an opt-in TCP flow inspector at the final dispatcher route:

- dispatcher-instance flow ID;
- source and routed destination;
- final outbound handler tag;
- logical uplink and downlink byte counters;
- active and closed lifecycle snapshots.

Tracking is disabled by default. UDP identity, mux substream identity, external
API transport, persistence, Android integration, and production readiness are
not implemented.

See [the PoC contract](docs/TCP_FLOW_TRACKER_POC.md),
[the exact upstream baseline](docs/FORK_BASELINE.md), and
[the active roadmap](docs/ROADMAP.md).

## Validate

```text
go test ./transport/pipe ./features/routing ./app/dispatcher
go test -race ./app/dispatcher
go vet ./transport/pipe ./features/routing ./app/dispatcher
```

The focused checks run on Linux and Windows in GitHub Actions. Some broader
upstream integration tests require external network responses and geodata assets;
their current boundary is documented in the PoC notes.

## Contributing

Issues and pull requests are welcome. Keep changes bounded, include tests,
preserve upstream behavior outside the changed boundary, and clearly distinguish
host tests from Android, device, network, and release evidence. See
[CONTRIBUTING.md](CONTRIBUTING.md).

## Upstream and license

Upstream source and project documentation belong to
[XTLS/Xray-core](https://github.com/XTLS/Xray-core). This fork retains the
upstream Mozilla Public License 2.0 in [LICENSE](LICENSE). See
[LICENSING.md](LICENSING.md) for a plain-language distribution summary.
