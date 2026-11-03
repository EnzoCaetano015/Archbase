# Architecture Starter and skill delivery

Architecture Starter is a free MIT skill for architecture onboarding: it identifies
the project stack and organization, explains applicable local/catalog conventions,
and suggests evidence-based next steps. It is not a comprehensive audit. It does
not require `arc`, Go, network access or an Archbase MCP server at runtime.

## Install and use

The free [Archbase Architecture Starter listing on Agensi](https://www.agensi.io/skills/archbase-architecture-starter)
is approved and live. For that delivery method, connect your agent through
[Agensi's sign-in flow](https://www.agensi.io/docs/mcp-setup), then ask it to use
Archbase Architecture Starter for a bounded project or supplied snapshot.
Instructions and requested references are delivered as text through Agensi MCP;
this does not install the CLI or a persistent skill directory. Publication alone
does not certify every agent or replace end-to-end delivery testing.

For a separate local MIT installation:

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

The 0.2.0 package adds focused references for optional CLI/MCP assistance and
safety/consent. Resolve the actual project before querying tools. A connected MCP
server must be confirmed to use that same root; missing tools use bundled text,
while invalid local configuration still blocks affected conclusions. CLI SemVer
alone does not identify the catalog commit, and queries never authorize setup,
updates, writes, permission repair or an extra data-processing service.

For a local MIT archive, `.cursor/skills/archbase-architecture-starter`,
`.claude/skills/archbase-architecture-starter` and supported Copilot surfaces'
`.github/skills/archbase-architecture-starter` are documented project locations.
Check your host's current discovery rules and tested compatibility before relying
on them: [Cursor](https://cursor.com/docs/skills),
[Claude Code](https://code.claude.com/docs/en/skills),
[Copilot](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills).
These recipes do not certify those agents. Extract the generated ZIP with SKILL.md
directly inside the named directory; preserve existing files until replacement
is explicitly chosen. No runtime script or binary is shipped.

The approved listing describes skill content version **0.2.0**. Local ZIPs retain
their independent version and checksum verification. Agensi's account, connection
and provider requirements are separate from this text-only skill, which needs no
Python runtime or scripts. See the platform's
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

Execution failures use `status: error` and `errorReason` in the result contract.
They retain artifact/fixture identity, a timestamp and any available transcript,
without inventing a model response or assertion review. Errors block release just
like failed or unexecuted cases. Existing v1 result records remain valid.

Fixture sources use typed Markdown (`page.tsx.md`, `package.json.md`) instead of
incomplete compiler inputs. The loader extracts code and preserves logical project
paths for evidence and hashes. Markdown project files receive an additional suffix
too: `AGENTS.md.md` represents `AGENTS.md`. See the
[fixture guide](../internal/skillbundle/testdata/starter/README.md).
`eval-fixture -suite <suite.json> -scenario <id> -output <new-directory>` materializes
real project filenames only when explicitly requested, outside versioned fixtures,
without overwriting or executing anything. Supply that extracted project to the
evaluating agent. Hashes cover decoded contents and paths, not Markdown wrappers.
