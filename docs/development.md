# Development and contributions

Building Archbase requires Go 1.26 or newer. Keep Go packages under `internal/` until a public API is intentionally designed.

## Build and validate

```bash
git clone https://github.com/EnzoCaetano015/Archbase.git
cd Archbase
go test ./...
go vet ./...
go build -trimpath ./cmd/arc
./arc help
```

On Windows, run the binary with `.\arc.exe help`. Source builds report `arc dev` by default. Installation options are documented in the [installation guide](installation.md).

## Contribute

Create a branch, make the change, and run the validation commands above. Open a pull request describing the problem, resulting behavior, relevant tests, and documentation changes.

Follow the repository's [contribution rules](../AGENTS.md). In particular:

- Keep public YAML contracts, [schemas](schemas.md), tests, and documentation synchronized.
- Keep registry entries sorted by ID and validate every declared required file.
- Keep patterns focused on code structure and rules focused on architecture.
- Preserve local customization and require explicit opt-in before overwriting existing files.

## Release builds

To inject a version into a local build:

```bash
go build \
  -ldflags "-X github.com/EnzoCaetano015/Archbase/internal/version.Value=0.3.2" \
  ./cmd/arc
```

Official release archives are produced by the [release workflow](../.github/workflows/release.yml). A local versioned build alone does not produce the complete release assets.

## Publish a stable release

Stable versions use `vMAJOR.MINOR.PATCH` tags. Open **Actions > Release > Run workflow** on GitHub, select `main`, and provide:

- `version`: the semantic tag, such as `v0.3.2`.
- `message`: the annotated tag message and introduction to the generated release notes.

The workflow runs tests, static analysis, and builds across its platform matrix. Before publication, it also checks Windows version metadata, target archives, checksums, reproducibility, the injected CLI version, and the installer contract.

An existing release is rejected. An existing tag is accepted only when it points to the selected commit; otherwise the workflow fails. If the tag is absent, the workflow creates it after verification. Publication requires the verified tag.

Unsigned archives and checksums must be byte-for-byte reproducible for identical source, version, and timestamp inputs. Public checksums must describe the final delivered artifacts.

Linux is the official distribution channel. macOS and Windows artifacts remain unsigned previews. Signing preparation is inactive; see the [code signing policy](code-signing-policy.md).
