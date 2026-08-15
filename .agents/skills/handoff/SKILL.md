---
name: handoff
description: Create a compact continuation note for MXR Core using existing files, diffs, exact baseline, and validation evidence.
---

# Handoff

Include:

1. user objective, exact repository root, branch, upstream baseline SHA, remotes;
2. completed changes linked to owning code and fork documents;
3. exact validation commands and results on current bytes;
4. unresolved bugs, upstream conflicts, performance, Android/device, privacy,
   licensing, publication, and release gates;
5. safest next action and exact expected write-set;
6. sensitive, destructive, remote, or automatic actions that must not run.

Reference existing artifacts instead of duplicating them. Do not treat stale
notes, host unit tests, an emulator run, or a local commit as production, device,
Google Play, or publication evidence.
