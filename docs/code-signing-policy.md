# Code signing policy

Last updated: September 30, 2026.

Archbase does not currently have a code-signing certificate or an active signing integration. Linux is the official distribution channel. macOS and Windows artifacts remain available only as unsigned previews.

The SignPath Foundation application was not approved at the project's current level of public adoption and visibility. Archbase may reapply after gaining meaningful community adoption, sustained engagement, and independent external recognition. Until a future application or another trusted certificate is approved, the project will not display SignPath attribution, claim an affiliation, or describe any artifact as signed.

## Future signing scope

The repository retains inactive technical preparation for a possible future signing workflow. If trusted signing is approved, the initial scope will be limited to `arc.exe` inside the Windows AMD64 and ARM64 release ZIP archives. A release must originate exclusively from the protected GitHub Actions release workflow in this repository. Locally built binaries and artifacts from any other system will not be eligible for the project certificate.

The project is maintained individually by [Enzo Caetano](https://github.com/EnzoCaetano015), GitHub account `EnzoCaetano015`. This maintainer is the authorized committer, reviewer, and signing approver. Every signing request requires an explicit manual review and approval; automatic approval is not permitted.

## Release integrity

Current unsigned archives must pass the complete test, static-analysis, packaging, version-metadata, and reproducibility gates. They remain byte-for-byte reproducible for identical source, version, and timestamp inputs. SHA-256 checksums validate integrity, but do not establish publisher identity.

If signing is approved in the future, Authenticode signatures will be applied only after those gates. Signed executable bytes may differ between otherwise identical requests because the signature includes a trusted timestamp.

Public SHA-256 manifests must always be generated from the final artifacts delivered to users. After any future signing, the checksum manifest must be regenerated from the signed Windows ZIP archives and the unchanged non-Windows archives before publication. Previously published unsigned releases will not be modified or republished as signed artifacts.

## Reporting

Archbase does not collect telemetry or personal data; see the [privacy policy](privacy.md). Suspected certificate misuse, compromised release artifacts, or other security issues should be reported through [GitHub private vulnerability reporting](https://github.com/EnzoCaetano015/Archbase/security/advisories/new). General questions can be submitted through [GitHub Issues](https://github.com/EnzoCaetano015/Archbase/issues).
