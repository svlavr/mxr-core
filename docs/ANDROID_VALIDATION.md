# Android validation boundary

This fork currently has Go source and unit-test evidence only. It has no MXR Core
Android binding, `VpnService`, APK, device installation, signing, or Play release
evidence.

When an executable Android boundary is accepted, validate in increasing scope:

1. affected Go package tests and race tests on the host;
2. reproducible Android target build for the intended ABI/API contract;
3. binding or JNI contract tests;
4. one primary emulator scenario for the affected path;
5. physical-device, live-network, performance, background lifecycle, and release
   checks only when explicitly required.

Report each evidence class separately. A successful Go cross-compile is not an
Android runtime test, and an emulator run is not Play release evidence.
