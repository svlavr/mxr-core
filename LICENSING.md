# Licensing

This repository is based on XTLS/Xray-core and is distributed under the Mozilla
Public License 2.0 (`MPL-2.0`). The complete controlling license text is in
`LICENSE`. This summary is informational and is not legal advice.

## What MPL-2.0 allows

- use for private, commercial, and other purposes;
- modification and redistribution;
- distribution of executables under different terms, provided those terms do
  not restrict recipients' MPL rights to the covered source;
- combination with separately licensed open-source or proprietary files as a
  Larger Work.

## Obligations when distributing source

Covered source files, including modifications to covered files, must remain
available under MPL-2.0. Recipients must be told that MPL-2.0 applies and how to
obtain the license.

## Obligations when distributing executables

When an executable containing covered software is distributed outside an
organization, the corresponding covered source must be made available by a
reasonable and timely method. Recipients must be told where to obtain it. The
source offer must include modifications to covered files used to build that
executable.

## Separate application code

MPL-2.0 uses file-level copyleft. Separate files that contain no MPL-covered code
may be licensed differently, including under proprietary terms, even when they
are combined with covered files in a Larger Work. Copying covered source into a
nominally separate file can make that file covered.

## Repository practice

- Preserve the root `LICENSE` and upstream notices.
- Mark new covered Go files with `SPDX-License-Identifier: MPL-2.0`.
- Do not copy code from incompatible or unknown-license sources.
- Record the exact source revision used for every distributed binary.
- Before an app-store or binary release, verify that the public source matches
  the covered files used in the shipped build and include a source-location
  notice in the distribution.

Primary references:

- Mozilla Public License 2.0: https://www.mozilla.org/MPL/2.0/
- Mozilla MPL 2.0 FAQ: https://www.mozilla.org/en-US/MPL/2.0/FAQ/
- XTLS/Xray-core license: https://github.com/XTLS/Xray-core/blob/main/LICENSE
