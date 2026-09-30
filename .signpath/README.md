# SignPath release handoff

This directory prepares Archbase for SignPath Foundation review. It does not activate signing and contains no organization identifiers, project or policy slugs, API tokens, or secret placeholders.

Use `https://github.com/EnzoCaetano015/Archbase/blob/main/docs/installation.md` as the SignPath Foundation application Download URL. That public page contains the required attribution, download commands, security behavior, signing status, and uninstall instructions.

## Post-approval release flow

1. Run the complete release gate in GitHub Actions and build the unsigned archives twice. The two unsigned output directories must be byte-for-byte identical.
2. Upload `arc_vVERSION_windows_amd64.zip` and `arc_vVERSION_windows_arm64.zip` together inside one root ZIP artifact produced by GitHub Actions. These must be the exact unsigned ZIPs that passed the gate.
3. Submit that root artifact as one SignPath signing request using `artifact-configuration.xml`, with `version` set to `MAJOR.MINOR.PATCH` and `pe-version` set to `MAJOR.MINOR.PATCH.0`.
4. Manually review and approve the request. Automatic signing approval is not allowed by the project policy.
5. Download the signed result and recover both Windows ZIP archives without renaming them.
6. On Windows, verify each `arc.exe` Authenticode signature and trusted timestamp with `Get-AuthenticodeSignature`. Reject any status other than `Valid`.
7. Re-run the release smoke tests against each executable on a compatible architecture and verify all VERSIONINFO fields.
8. Regenerate `arc_vVERSION_SHA256SUMS.txt` from the final signed Windows ZIPs and the unchanged Linux and macOS archives.
9. Verify the final checksum manifest and publish only after every release gate passes.

The future GitHub integration will require the SignPath organization ID, project slug, signing policy slug, and a `SIGNPATH_API_TOKEN` secret. Add those values only after approval, in a separate reviewed change that uses the [official SignPath GitHub integration](https://docs.signpath.io/trusted-build-systems/github).
