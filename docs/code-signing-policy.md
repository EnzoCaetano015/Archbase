# Code signing policy

Last updated: September 30, 2026.

Free code signing provided by SignPath.io, certificate by SignPath Foundation.

Archbase is preparing its Windows release process for the SignPath Foundation open-source code-signing program. Signing is not active until the application is approved and the trusted build integration is completed. Release `v0.3.0` and every earlier release are unsigned and will not be modified or republished as signed artifacts.

## Signed artifact scope

The initial signing scope is limited to `arc.exe` inside the official Windows AMD64 and ARM64 release ZIP archives. A release must originate exclusively from the protected GitHub Actions release workflow in this repository. Locally built binaries and artifacts from any other system are not eligible for the project certificate.

The project is maintained individually by [Enzo Caetano](https://github.com/EnzoCaetano015), GitHub account `EnzoCaetano015`. This maintainer is the authorized committer, reviewer, and signing approver. Every signing request requires an explicit manual review and approval; automatic approval is not permitted.

## Release integrity

Unsigned inputs must pass the complete test, static-analysis, packaging, version-metadata, and reproducibility gates before a signing request is created. Authenticode signatures are applied only after those gates. Signed executable bytes may differ between otherwise identical requests because the signature includes a trusted timestamp.

Public SHA-256 manifests must always be generated from the final artifacts delivered to users. After signing, the checksum manifest is regenerated from the signed Windows ZIP archives and the unchanged non-Windows archives before publication.

## Reporting

Archbase does not collect telemetry or personal data; see the [privacy policy](privacy.md). Suspected certificate misuse, compromised release artifacts, or other security issues should be reported through [GitHub private vulnerability reporting](https://github.com/EnzoCaetano015/Archbase/security/advisories/new). General questions can be submitted through [GitHub Issues](https://github.com/EnzoCaetano015/Archbase/issues).
