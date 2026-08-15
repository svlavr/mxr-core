---
name: xray-fork-android-validation
description: Validate an affected executable Android boundary of MXR Core, such as cross-build, binding, native loading, TUN integration, lifecycle, packaging, ABI, or device runtime.
---

# Xray fork Android validation

1. Read `AGENTS.md`, `docs/ANDROID_VALIDATION.md`, and the affected binding or
   packaging contract. If no executable Android boundary exists, report that
   fact instead of fabricating device evidence.
2. Start with affected host tests and the exact Android target/ABI compile.
3. Validate binding symbols, native loading, packaging, and lifecycle before
   starting a VPN or live-network scenario.
4. Use one primary emulator only for an actually executable affected path. Test
   the changed scenario; do not automatically run a full API/ABI matrix.
5. Expand to API 29, additional ABIs, physical devices, live networks,
   performance, background lifecycle, signing, or store packaging only when the
   user or active roadmap explicitly requires it.
6. Store secrets and raw evidence only under ignored `.local/` paths. Redact
   addresses, credentials, node material, captures, and logs before sharing.
7. Stop repeating runs when no new signal is expected. Record the smallest next
   diagnostic and the exact blocker.
8. Report host source/tests, Android cross-build, emulator, physical device,
   live network, signing, and release evidence as separate classes.
