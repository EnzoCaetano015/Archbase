# Architecture Starter and skill delivery

Architecture Starter is a free MIT skill for architecture onboarding: it identifies
the project stack and organization, explains applicable local/catalog conventions,
and suggests evidence-based next steps. It is not a comprehensive audit. It does
not require `arc`, Go, network access or an Archbase MCP server at runtime.

## Install and use

Use the versioned Starter ZIP produced by the **Skills** workflow. Verify its
SHA-256 against `SHA256SUMS`, inspect its text, and extract it into a folder named
`archbase-architecture-starter` in your agent's supported skills directory. Agent
installation paths vary; use your agent's documented skill-loading method. The
ZIP has `SKILL.md` at its root. Do not install the raw source folder: generated
catalog references are materialized during packaging.

Ask: "Use Architecture Starter to explain this project's architecture and suggest
relevant Archbase conventions. Do not change files." The skill responds in your
language. It uses the supplied project root or invocation directory, preserves
local decisions, and distinguishes adopted rules from recommendations.

Agensi listing/submission is a separate publication step; this repository does
not claim an approved live listing. When delivered through Agensi, references are
loaded as text on demand via its MCP rather than installed on disk. See its
[packaging guide](https://www.agensi.io/docs/packaging).

## Build locally

Development uses the existing Go 1.26+ toolchain and Git. No additional language
runtime or dependency is required. Fetch the pinned public catalog tag first if
it is not already present; the tooling never fetches or installs implicitly.

```powershell
./scripts/verify-skills.ps1 -Output C:/temp/starter-delivery
```

On any supported Go platform, run the utility directly:

```bash
go run ./internal/skillbundle/cmd/skillbundle validate \
  -root skills/archbase-architecture-starter \
  -manifest skills/archbase-architecture-starter/bundle.json \
  -lock skills/catalog.lock.json -catalog-root .
go run ./internal/skillbundle/cmd/skillbundle package \
  -root skills/archbase-architecture-starter \
  -manifest skills/archbase-architecture-starter/bundle.json \
  -lock skills/catalog.lock.json -catalog-root . -output dist/starter-new
```

The destination's parent must exist, and the destination must be new. Packaging
creates the ZIP, checksum and `unpacked` inspection tree together. It never
overwrites a previous delivery. Failures roll back only that newly created output.

The catalog lock pins `v0.8.0` and its full commit, hashes the indices, MIT license,
canonical rules, referenced pattern manifests and required pattern files. Git
blobs are read at that commit, independent of dirty checkout files and CRLF
conversion. Generated Markdown preserves restrictions and pattern IDs; it does
not manually paraphrase canonical rules. Only selected rule references enter the
bundle; source examples are validated without being shipped unnecessarily.

To update the catalog, generate a new lock at a verified tag into a new path,
review it, replace the old lock deliberately, update selected files and rerun tests:

```bash
go run ./internal/skillbundle/cmd/skillbundle lock \
  -catalog-root . -tag v0.8.0 \
  -commit 240950368facf47684d574cd95b8e5132d3afdef \
  -output catalog-next.lock.json
```

## Contracts and verification

New JSON contracts have schemas under `schemas/skill-*.schema.json`. A manifest
declares skill name, version, license, every delivered path and sorted RuleIDs.
Frontmatter uses standard `name`, `description`, `license`, `metadata.version`.
The root directory, manifest and frontmatter must agree. See the source
[manifest](../skills/archbase-architecture-starter/bundle.json).

The conservative delivery profile permits at most 100 files, 5,000,000 total text
bytes and 1,000,000 bytes per file. Scripts are limited to 20 KB and are optional;
the Starter has none. Only allowlisted UTF-8 text types enter the package. Local
links must resolve within it; hidden/development files, executables, symlinks,
path escapes, case-insensitive collisions and known credential patterns fail.
Secret scanning is a heuristic and does not replace review of deliverable text.

ZIP entries use sorted paths, fixed 1980 timestamps, mode 0644, normalized LF text
and stored bytes. CI verifies repeated builds, extraction and checksums on Windows
and Linux, then compares their digests. This workflow uploads artifacts for review;
it does not publish releases or submit to Agensi.

Behavioral evaluations are separate from package checks. The development utility
validates scenario and result schemas, fixtures, version identities, evidence
hashes, transcripts and supervised assertion reviews. `not-run` is explicit and
blocks `eval-check -release`. A human/local evaluator must assess the transcript;
the validator does not claim to infer model correctness from keywords or booleans.
The Premium repository tracks its 30-scenario suite; OSS example fixtures and
unit/integration tests remain independently executable here.

Fixture sources use typed Markdown (`page.tsx.md`, `package.json.md`) instead of
incomplete compiler inputs. The loader extracts code and preserves logical project
paths for evidence and hashes. Markdown project files receive an additional suffix
too: `AGENTS.md.md` represents `AGENTS.md`. See the
[fixture guide](../internal/skillbundle/testdata/starter/README.md).
`eval-fixture -suite <suite.json> -scenario <id> -output <new-directory>` materializes
real project filenames only when explicitly requested, outside versioned fixtures,
without overwriting or executing anything. Supply that extracted project to the
evaluating agent. Hashes cover decoded contents and paths, not Markdown wrappers.
