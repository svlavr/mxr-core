# Android core artifact validation

This fork currently has Go source and host-test evidence only. It has no validated
Android core library or binding.

When an Android core artifact is accepted, validate in increasing scope:

1. affected Go package tests and race tests on the host;
2. reproducible Android cross-build for the intended ABI and API contract;
3. binding or JNI contract tests;
4. process-local load, configure, start, call, cancel, stop, and native-resource
   cleanup checks;
5. emulator or physical-device checks only when required to prove core-artifact
   behavior.

Report each evidence class separately. A successful cross-build is not runtime
validation.
