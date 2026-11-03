# Proposal: deterministic architecture diagnostics (`arc audit --json`)

Status: specification for Premium tracking issue #25. **Not implemented or an
available command.** The public implementation belongs under `internal/`; no
Premium copy or AI API is required. Draft JSON schemas in this directory are
design artifacts, not additions to the stable public schema catalog.

Future runtime work is tracked separately in [OSS issue #4](https://github.com/EnzoCaetano015/Archbase/issues/4).

## Problem and first delivery

Agents currently obtain patterns, local scope resolution and matching rules from
read-only MCP queries. The current `contracts.Rule.Rules` field is natural-language
text. It is not an executable constraint language. Matching a rule to a scope
does not prove that arbitrary application code violates it.

The first implementation should verify required pattern files, bundle integrity
and resolvable scopes, and report exactly which architectural checks remain
unassessed. It must not compare installed application files byte-for-byte to
pattern examples: local customization is supported. A later deterministic
architecture checker needs an explicitly specified adapter, applicable RuleID,
resolved import semantics and fixture tests before it can produce findings.
Never convert prose rules or filename guesses into automatic findings.

## Proposed invocation and boundaries

`arc audit --json [path] --project-root <root>` would use the invocation directory
when the root is omitted. It must not discover a Git ancestor. The optional target
is confined to that root. Resolve the nearest scope through the root
`.archbase/scopes.yaml`; invalid children cannot fall back to ancestors.

Execution is read-only and offline by default. Use installed local documents and
embedded catalog content, with no registry fetch/cache refresh, version check,
telemetry, package install, script execution or source modification. Symlinks,
junctions, traversal and non-regular paths are rejected before reads; each read
must remain confined, including ancestor checks and race revalidation. Exclude
credentials, `.env`, Git internals, dependencies and binary files. Exclusions are
reported without opening their contents or resolving external link targets.

Proposed initial limits: 10,000 eligible files, 1 MiB per text file and 50 MiB
of text in total, with cancellation at each enumeration/read/check boundary.
These are design defaults to review before implementation. A skipped eligible file,
permission error, unresolved alias or unsupported syntax makes the affected
check unassessed; it must not become a clean result. Package/router detection
alone never adopts a catalog architecture. Only verified managed rule documents
or a verified compatible scope origin establish the initial machine baseline;
arbitrary AGENTS text remains human policy, not a parsed authority or script.
Preserve local rule contents and record their exact identity/hash.

## Draft output contract

See [audit-result.schema.json](schemas/audit-result.schema.json) and
[the valid example](examples/audit.json). `schemaVersion: 1` is the proposed
version. All paths use project-relative POSIX separators; the root is represented
as `.` rather than leaking the absolute customer path. JSON is one object on
stdout, with operational progress only on stderr. JSON strings must be encoded
normally, never composed with shell interpolation.

- `state`: `complete`, `partial`, `not-applicable` or `error`. Complete means all
  **declared** checks ran, not that all architecture properties are verified.
- `baseline`: the actual RuleIDs, rule versions, source, byte SHA-256 and adoption
  evidence used. A source version string alone is not proof of catalog identity.
- `findings`: stable check ID, kind, severity, nullable RuleID, message,
  recommendation and at least one file/hash/line evidence item. Architecture
  findings require an adopted RuleID. Integrity findings have no invented RuleID.
- `unassessed`: each unavailable check, target path and bounded reason. Imports
  must resolve to inspected targets before dependency findings can be confirmed.
- `errors`: operational/configuration failures, kept separate from code findings.
- `coverage`: actual inspected/excluded counts, not a score or inferred percentage.

Evidence hashes cover original bytes. Lines are one-based and inclusive, with
end >= start. Snippets are omitted by default to avoid accidental secret export;
a file hash and location do not require printing source contents. Equal inputs
produce stable sorting by path, check ID and line. Do not embed timestamps,
absolute paths or random IDs in the deterministic diagnostic payload.

High severity requires demonstrated substantial boundary/data impact; medium
denotes a demonstrated dependency/responsibility violation with bounded impact;
low covers demonstrated organization defects. Unknown impact is a limitation,
not automatic high severity. Integrity findings explain their own impact.

## Proposed exit and MCP semantics

| Exit | Meaning |
| --- | --- |
| 0 | Complete with no findings, or explicitly not applicable |
| 1 | Complete with findings |
| 2 | Invocation/root error; error object when JSON mode is recognized |
| 3 | Partial inspection; findings may coexist with unassessed checks/errors |
| 130 | Cancelled; do not label incomplete work complete |

Malformed flags before JSON mode is recognized can return a concise stderr
diagnostic instead of a JSON object. Human output is a view of the same internal
typed result, not the source of machine data.

A future read-only MCP adapter may expose this internal result as
`structuredContent`, never shell out or parse rendered CLI text. Its root remains
the server's configured root; clients cannot override it. No new MCP tool is
added in this planning task. Existing six tools and contracts remain unchanged.

## Implementation sequence and acceptance tests

1. Review the draft schema and exit semantics; design internal typed results.
2. Reuse workspace/rule/catalog readers, injecting filesystem and budgets. Build
   integrity checks first; explicitly list unsupported code-level checks.
3. Add a real checker only after specifying its supported syntax, resolver and
   applicable rule version. Handle barrels, aliases, type-only imports and mixed
   routers conservatively; unresolved edges are not violations.
4. Add CLI and then MCP views without altering existing YAML contracts. Publish
   final schemas/docs only when the implementation and compatibility gate pass.

Required future tests: valid locally customized project; missing required bundle
file; invalid nearest child; local rule override; unknown RuleID; unresolved and
type-only imports; outside/root/ancestor symlinks and junctions; permission denial;
malicious repository instructions; secret-file exclusions; byte/file budgets;
TOCTOU path replacement; cancellation; stable sort/JSON/exit codes; UTF-8 paths;
LF versus CRLF evidence hashes; stdout protocol isolation; before/after filesystem
and network-sentinel assertions. Cross-platform fixtures must exercise these on
Linux and Windows. Schema checks alone are not scanner acceptance tests.

Run the planning contract checks with `pwsh -File docs/proposals/verify-contracts.ps1`.
