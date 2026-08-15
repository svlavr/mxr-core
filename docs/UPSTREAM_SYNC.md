# Upstream synchronization

## Remotes

- `upstream`: official `XTLS/Xray-core`; fetch only, never a publication target.
- `origin`: the MXR Core repository. Do not change its visibility without
  explicit authorization.

## Safe synchronization sequence

```text
git status --short
git fetch --prune upstream
git log -1 --format="%H %cI %s" upstream/main
go test ./app/dispatcher
```

Create a dedicated synchronization branch from the current fork branch. Rebase
or merge only after reviewing upstream changes that touch fork-owned files.
Never force-push a shared branch. After resolving conflicts, run affected tests,
the race test for the tracker, and any broader build gate required by the touched
upstream packages. Update `docs/FORK_BASELINE.md` only when the synchronized
commit is actually accepted.

## Expected conflict surface

The current patch intentionally touches a small surface:

- `app/dispatcher/default.go` — final outbound selection hook;
- `app/dispatcher/tcp_flow_tracker.go` — fork implementation;
- `features/routing/tcp_flow.go` — fork-facing inspection contract;
- `app/dispatcher/tcp_flow_tracker_test.go` — bounded regression tests.
- `transport/pipe/impl.go`, `reader.go`, and `writer.go` — optional byte callbacks
  that preserve concrete pipe types used by mux and outbound fast paths.

If upstream changes dispatcher lifecycle or `transport.Link` ownership, treat
the tracker semantics as unverified until the tests and review are updated.
