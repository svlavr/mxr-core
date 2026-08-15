# Fork baseline

## Source of truth

- Upstream repository: `https://github.com/XTLS/Xray-core.git`
- Local remote name: `upstream`
- Baseline branch: official `main`
- Baseline commit: `7d214f8b094f75322fa3990f8aadad1c912f24f5`
- Upstream commit time: `2026-08-12T08:38:01Z`
- Upstream subject: `WireGuard outbound: Fix sendThrough support (#6570)`
- Verified on: `2026-08-15`

This baseline was cloned directly from official GitHub. It did not reuse an
older local Xray checkout.

## Provenance and license

The upstream repository is licensed under MPL-2.0; the root `LICENSE` remains
the controlling inherited license text. Fork modifications must retain upstream
notices and must be reviewed for MPL source-distribution obligations before an
MXR Core library or executable is published.

## Baseline update rule

Do not write "latest" without an exact commit. Before changing this baseline:

1. fetch `upstream` from the official URL;
2. inspect the official `main` commit and relevant tags/releases;
3. record the chosen SHA and UTC commit time here;
4. run the fork test gates before and after synchronization;
5. document conflicts and intentional deviations.

GitHub's `releases/latest` marker is not used as the sole freshness signal;
official refs and the `upstream/main` commit are checked directly.
