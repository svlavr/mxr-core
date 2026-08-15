---
name: security-threat-model
description: Build a repository-grounded threat model for MXR Core when the user explicitly requests threats, abuse paths, or AppSec modeling.
---

# Xray fork threat model

Create a concise AppSec threat model grounded in exact repository paths. Do not
trigger for ordinary architecture explanation or code review.

1. Define in-scope paths, deployment assumptions, entrypoints, and exclusions.
   Separate upstream code, fork changes, build/CI, tests, Android bindings, and
   application integration.
2. Map trust boundaries for network inputs, configurations, control/API clients,
   DNS/routing, outbound transports, local files, secrets, native/platform
   integration, build provenance, and update/publication channels when present.
3. List assets and realistic attacker capabilities/non-capabilities. Never expose
   discovered secrets; only describe and redact their location.
4. Write a small set of concrete multi-step abuse paths tied to evidence. Rate
   likelihood and impact with explicit assumptions and existing controls.
5. Before finalizing, ask the user 1–3 questions only when deployment, exposure,
   privilege, multi-user use, or sensitive data would materially change rank.
6. Tie mitigations to exact components and distinguish existing controls from
   recommendations. Include detection and residual risk for high items.
7. Produce one compact Mermaid data-flow diagram and save the report as
   `docs/security/<scope>-threat-model.md`.

Use `references/output-contract.md` for the final structure. Prioritize remote
code execution, config/parser abuse, routing or DNS integrity compromise, secret
exposure, control-API authorization, resource exhaustion, unsafe native/binding
boundaries, and supply-chain/release compromise only where evidence supports it.
