# Diagnose local configuration with `arc doctor`

`arc doctor` inspects Archbase configuration without repairing or changing it.
It is implemented in the source tree; use a source build or a published CLI
release that includes this command. Older releases may not recognize `doctor`.
This is a configuration diagnostic, not an application architecture auditor.

```bash
arc doctor
arc doctor --verbose
arc doctor --json
arc doctor --project-root /absolute/path/to/project --json
```

The invocation directory is the default root. The command does not search for a
Git root, initialize `.archbase`, refresh a registry, check for updates, install
dependencies, execute project scripts or change permissions. Global registry
flags do not enable network access for this command. No MCP tool is added.

## Checks

Every run returns these ten unique checks, sorted by ID:

| Check | What is inspected |
| --- | --- |
| `doctor.cli-version` | The invoked binary's version, including `dev`; no latest-release lookup |
| `doctor.project-root` | A regular, readable root with safe ancestors |
| `doctor.workspace-index` | The existing root `.archbase/scopes.yaml` contract |
| `doctor.scope-paths` | Sorted, unique, confined directory scopes; no invalid-child fallback |
| `doctor.local-bundles` | Local identity, manifests, typed examples and all declared required files |
| `doctor.scope-origin` | Origin metadata; exact embedded origins verified offline where available |
| `doctor.rule-documents` | Owned central documents, matching metadata, IDs and filenames |
| `doctor.managed-indices` | Managed AGENTS/Claude/Cursor/Copilot blocks against owned documents |
| `doctor.legacy-indices` | Unsupported per-directory `.archbase/scope.yaml`, without loading or migrating it |
| `doctor.snapshot` | Stable identities and metadata of inspected files and directories |

An absent workspace is a warning, not a broken application. With no local
bundles, origins or active managed indices, the respective checks are
`not-applicable` with reasons. Malformed configuration blocks dependent checks
as `not-run`; the command never hides an invalid child scope behind its parent.
Missing optional files are allowed. Local example customization is not compared
byte-for-byte with the catalog. A missing optional origin is not corruption.
An unknown embedded origin is invalid; other registries and unavailable origin
versions remain explicitly unverified offline rather than triggering a fetch.

Foreign central Markdown is not claimed as owned. A managed index referencing
it cannot pass the owned-reference comparison. External instruction bytes are
preserved; Cursor frontmatter is checked only for an active managed index.

## Human output, JSON and exit codes

Interactive human output shows ten friendly check names with status symbols,
counts and coverage. Partial and cancelled runs remain explicitly incomplete.
`--verbose` (`-v`) adds IDs, paths, full messages, reasons and recommendations.
A delayed spinner indicates work in progress. Plain and redirected output retain
the original detailed format. JSON takes precedence over verbose and never animates.
`--json` writes exactly one versioned object to stdout; a
diagnostic finding is not printed again as an error on stderr. Command/flag
parsing failures may use stderr before the diagnostic starts.

The stable runtime contract is
[`schemas/doctor-result.schema.json`](../schemas/doctor-result.schema.json).
Its fields are `schemaVersion: 1`, `command: "doctor"`, `toolVersion`,
`projectRoot: "."`, `state` and `checks`. Check fields are `id`, `status`, nullable
`path`, `message`, and optional `reason` and `recommendation`. Every non-pass
check requires a reason. Paths are relative POSIX paths, or null; no timestamps,
absolute roots, file excerpts or configuration values are emitted.

| Exit | Meaning |
| --- | --- |
| 0 | Complete, without configuration errors; warnings and not-applicable checks are explicit |
| 1 | Complete with configuration errors |
| 2 | Invocation/root failure |
| 3 | Partial, including blocked dependent checks; takes precedence over exit 1 |
| 130 | Cancelled; takes precedence over other diagnostic exits |

`complete` describes inspection coverage, not that the configuration is valid.
`partial` requires at least one `not-run` check. `error` means an invalid invocation
or an unavailable root. Neither a partial run nor a cancellation is success.

## Read boundaries and limits

Reads are anchored to OS directory handles. Traversal, symlinks and Windows
junctions in the selected root, its ancestors or inspected paths are rejected.
Use the real directory path when an ancestor is a symlink. Concurrent identity
or metadata changes invalidate snapshot verification. This is not a filesystem
freeze: other processes can still change files, and metadata checks cannot
detect every possible concurrent content edit that restores the same metadata.

Inspection is bounded to 10,000 directory entries/file opens, 1 MiB per file and
50 MiB of file reads in total. Directory listings are cached only while their
identity and metadata remain unchanged. Exhaustion and permission denial become
`not-run`, never silent truncation. Cancellation is checked between reads.

Only expected metadata, declared bundle files and native instruction files are
read. Known credential/environment paths (including `.env*`, private keys,
credential stores and secret directories) are excluded before opening them.
Diagnostics never echo parser errors containing source/configuration values.
Do not use these exclusions as a substitute for keeping secrets out of patterns.

The legacy scan examines directory metadata and skips dependency/build/cache
directories, sensitive directories and nested Git projects. It does not follow
unrelated links or read application source. Finding no legacy configuration in
that inspected tree is not a claim about excluded areas.

Nothing is repaired, removed, migrated or sent to an external service. Review
recommendations and authorize any later edits separately. Deterministic
application-code audits remain tracked in [issue #4](https://github.com/EnzoCaetano015/Archbase/issues/4).
