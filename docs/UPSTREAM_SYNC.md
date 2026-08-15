# Upstream synchronization

## Remotes

- `upstream`: official `XTLS/Xray-core`; fetch only, never a publication target.
- `origin`: the MXR Core repository. Do not change its visibility implicitly.

## Safe synchronization sequence

```text
git status --short
git fetch --prune upstream
git log -1 --format="%H %cI %s" upstream/main
go test ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher
go test -race ./common/session ./common/mux ./transport/pipe ./features/routing ./app/dispatcher
```

Create a dedicated synchronization branch from the current fork `main`. Rebase
or merge only after reviewing upstream changes that touch the fork extension.
Never force-push a shared branch. Update `FORK_BASELINE.md` only when the
synchronized commit is accepted and the affected checks pass before and after
the synchronization.

## Expected conflict surface

- `app/dispatcher` — final outbound selection, tracker implementation, tests,
  and lifecycle benchmarks;
- `features/routing/tcp_flow.go` — fork inspection contract;
- `common/session` — outbound error feedback attached to a tracked flow;
- `common/mux` — logical session and physical worker termination semantics;
- `transport/pipe` — exact-final-counter detach barrier and byte accounting.

If upstream changes dispatcher lifecycle, MUX termination, error feedback,
`transport.Link` ownership, or pipe synchronization, treat the tracker contract
as unverified until its focused tests, race tests, and benchmarks are repeated.
