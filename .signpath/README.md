# Future SignPath reapplication preparation

The current SignPath Foundation application was not approved because Archbase does not yet have the required level of public adoption and external recognition. This directory is retained only as inactive technical preparation for a possible future reapplication. It does not activate signing, establish an affiliation with SignPath, or contain organization identifiers, project or policy slugs, API tokens, or secret placeholders.

Reapply only after the project has meaningful public adoption, sustained activity, and independent references. Until approval, Windows releases remain unsigned previews and no release may be described as SignPath-signed.

## Conditional flow after a future approval

1. Run the complete release gate in GitHub Actions and build the unsigned archives twice. The two unsigned output directories must be byte-for-byte identical.
2. Upload `arc_vVERSION_windows_amd64.zip` and `arc_vVERSION_windows_arm64.zip` together inside one root ZIP artifact produced by GitHub Actions. These must be the exact unsigned ZIPs that passed the gate.
3. Submit that root artifact as one SignPath signing request using `artifact-configuration.xml`, with `version` set to `MAJOR.MINOR.PATCH` and `pe-version` set to `MAJOR.MINOR.PATCH.0`.
4. Manually review and approve the request. Automatic signing approval is not allowed by the project policy.
5. Download the signed result and recover both Windows ZIP archives without renaming them.
6. On Windows, verify each `arc.exe` Authenticode signature and trusted timestamp with `Get-AuthenticodeSignature`. Reject any status other than `Valid`.
7. Re-run the release smoke tests against each executable on a compatible architecture and verify all VERSIONINFO fields.
8. Regenerate `arc_vVERSION_SHA256SUMS.txt` from the final signed Windows ZIPs and the unchanged Linux and macOS archives.
9. Verify the final checksum manifest and publish only after every release gate passes.

A future integration may add the approved provider's identifiers and secrets only in a separate reviewed change after approval. Until then, keep `artifact-configuration.xml` dormant and do not add signing actions or credential placeholders.
