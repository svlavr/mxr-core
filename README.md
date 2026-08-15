# MXR Core

MXR Core is an experimental fork of
[XTLS/Xray-core](https://github.com/XTLS/Xray-core). The fork keeps upstream
history and focuses on small, reviewable runtime experiments.

## Current experiment

The first change is an opt-in TCP flow inspector at the final dispatcher route:

- dispatcher-instance flow ID;
- source and routed destination;
- final outbound handler tag;
- exact final uplink and downlink byte counters;
- `completed`, `cancelled`, and `failed` completion reasons;
- logical MUX substream lifecycle;
- active and closed lifecycle snapshots.

The experiment has a `KEEP` decision as an internal opt-in capability and remains
disabled by default. It does not expose a stable external core API. UDP identity,
event transport, persistence, Android binding, and production readiness are not
implemented.

See [the PoC contract](docs/TCP_FLOW_TRACKER_POC.md),
[the exact upstream baseline](docs/FORK_BASELINE.md), and
[the active roadmap](docs/ROADMAP.md).

## Validate

```text
go test ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher
go test -race ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher
go vet ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher
```

The focused checks run on Linux and Windows in GitHub Actions. Some broader
upstream integration tests require external network responses and geodata assets;
their current boundary is documented in the PoC notes.

## Contributing

Issues and pull requests are welcome. Keep changes bounded, include tests,
preserve upstream behavior outside the changed boundary, and clearly distinguish
source, build, live-network, and release evidence. See
[CONTRIBUTING.md](CONTRIBUTING.md).

## Upstream and license

Upstream source and project documentation belong to
[XTLS/Xray-core](https://github.com/XTLS/Xray-core). This fork retains the
upstream Mozilla Public License 2.0 in [LICENSE](LICENSE). See
[LICENSING.md](LICENSING.md) for a plain-language distribution summary.
