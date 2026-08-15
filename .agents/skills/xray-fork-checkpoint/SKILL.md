---
name: xray-fork-checkpoint
description: Safely checkpoint a completed verified MXR Core slice without creating remotes, changing visibility, pushing upstream, or publishing unapproved artifacts.
---

# Xray fork checkpoint

This skill may stage and commit only a completed, owned slice. Publication is
allowed only to an existing authorized `origin`; `upstream` is never writable.

## Gates

1. Read `AGENTS.md`, current branch/status/diffs, pre-existing index, and the
   declared write-set. Preserve unrelated changes and require an empty unrelated
   index.
2. Require an accepted feature branch (`codex/*` or another explicitly named
   branch) and passing current-byte tests. Never commit a failed experiment as a
   completed slice.
3. Scan exact candidates for credentials, endpoints, signing material, captures,
   binaries, archives, `.local/`, logs, and unexpected generated or large files.
4. Stage explicit paths only. Never use `git add .`, `git add -A`, or broad globs
   in a mixed tree. Inspect the full staged name list, stat, diff, and
   `git diff --cached --check`.
5. A local commit may be created only when the user asked to save/checkpoint or
   project policy explicitly authorized it. Use one conventional commit per
   coherent slice.
6. Push only when the user asked to publish and `origin` already exists with the
   intended target. Never create GitHub repositories, set private/public
   visibility, push to `upstream`, force-push, tag, release, or publish binaries.

Report `PUBLISHED`, `LOCAL_ONLY`, `NO_CHECKPOINT`, or `BLOCKED`, with commit SHA,
branch, validation, exclusions, and exact remediation.
