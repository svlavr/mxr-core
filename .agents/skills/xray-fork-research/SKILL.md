---
name: xray-fork-research
description: Evidence-first research for Xray fork divergence, upstream behavior, comparable clients, licensing, Android integration, security, and publication decisions.
---

# Xray fork research

Turn a question into one bounded verdict without automatically writing code.

## Modes

- `QUICK_ANSWER`: repository and official upstream already decide the question.
- `COMPARISON`: two or more real alternatives must be compared.
- `FORK_DECISION`: a new divergence or external contract needs a durable record.
- `EXPERIMENT_REQUIRED`: evidence cannot decide; define the smallest test.

## Workflow

1. State the underlying need and neutral, testable questions.
2. Read `AGENTS.md`, `docs/FORK_BASELINE.md`, the relevant fork code/tests, and
   the exact official upstream code before looking at downloaded clients.
3. Keep a ledger of `KNOWN`, `DISPUTED`, and `UNKNOWN` facts. Record what would
   change the verdict.
4. Verify official GitHub refs and exact SHAs. Do not equate a stale
   `releases/latest` marker, popularity, or another client's UI with core truth.
5. For each alternative record version/ref/date, license, observed behavior,
   reusable role, edge cases, and prohibited reuse.
6. Treat external and downloaded clients as read-only evidence. Never import
   GPL, proprietary, or no-license implementation, resources, names, or control
   flow into this fork.
7. Stop once the gap is stable or a bounded experiment is the only discriminator.

Use `references/evidence-and-output.md` for verdicts and response shape. Prefer
plain Russian unless the user requests another language. Keep source/build,
synthetic, Android, physical-device, production, legal, and release evidence
separate.
