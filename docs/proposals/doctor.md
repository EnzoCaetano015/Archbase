# Proposal: local configuration diagnostics (`arc doctor`)

Status: historical specification for Premium tracking issue #26. The read-only
runtime is now implemented in the public source tree under `internal/diagnostics`.
See [the runtime guide](../doctor.md) and [final v1 result schema](../../schemas/doctor-result.schema.json).
The draft below is preserved as design history; it does not replace the final
contract. Use a source build or a published release containing the command.

Future runtime work is tracked separately in [OSS issue #5](https://github.com/EnzoCaetano015/Archbase/issues/5).

## Proposed behavior

`arc doctor [--json] [--project-root <root>]` would inspect the invocation directory
by default, without Git-root discovery. It answers whether local Archbase
configuration is readable, internally consistent and confined to the selected
project. It does not audit application architecture or contact external services.

| Check ID | Planned verification |
| --- | --- |
| `doctor.cli-version` | Report the invoked executable's actual version, including dev builds; no latest-release lookup |
| `doctor.project-root` | A regular, readable directory, including safe ancestors |
| `doctor.workspace-index` | Root `.archbase/scopes.yaml` parses under the supported contract; absence is an unconfigured warning |
| `doctor.scope-paths` | Unique normalized in-root scopes; closest child invalidity is visible |
| `doctor.local-bundles` | Local roots remain inside `.archbase`; manifest identity and required declared files validate |
| `doctor.scope-origin` | Origin mapping is valid when declared; customization is not corruption |
| `doctor.rule-documents` | Installed owned central rule documents are readable, valid and unambiguous; duplicates/collisions are surfaced |
| `doctor.managed-indices` | Active AGENTS/Claude/Cursor/Copilot managed references agree with owned documents; preserve external bytes |
| `doctor.legacy-indices` | Report legacy per-directory scope files as unsupported configuration, without loading or migrating them |

Each check reports `passed`, `warning`, `error`, `not-applicable` or `not-run`, a stable code,
project-relative nullable path, explanation and non-executed recommendation.
Permissions/configuration errors block dependent checks, which become `not-run`
with a reason; they cannot pass through an ancestor fallback. Root failure must
not prevent an already obtainable CLI version from being reported, but no project
read occurs outside the authorized root.

An absent root index is a warning rather than proof that the application is
broken; bundle checks with no configured bundles are `not-applicable`, with a
reason, rather than falsely passing or blocking the whole run. A malformed
root/child index is an error and makes dependent checks `not-run`. A missing optional file is not
equivalent to a missing required file. A missing origin on a local custom pattern
is not automatically an error. No exact-match comparison with remote examples.
Managed index validation never infers ownership/deletion authority from links.

## Safety and output

Offline, read-only operation: no remote registry refresh, cache rewrite, install,
PATH change, version update, project initialization, legacy cleanup, permission
repair or script execution. Diagnostic text can propose a later explicit action;
it cannot perform that action. Do not read credential/environment files or print
source/configuration values that may contain secrets. Symlink/junction targets
outside the boundary must not be followed or disclosed.

Use the same proposed 10,000-file / 1 MiB-per-file / 50 MiB-total text budgets as
the audit proposal, with cancellation between reads. Exhausted checks become
`not-run` and the overall result becomes partial; never truncate into success.

The draft [doctor-result schema](schemas/doctor-result.schema.json) and
[example](examples/doctor.json) define a proposed versioned JSON shape. Project
root is represented by `.`, with relative POSIX paths and deterministic ordering.
Do not put timestamps or absolute paths in the payload. `checks` is nonempty and
unique by ID. Non-pass checks require a reason; recommendations are text only.
State `complete` means planned checks ran even when they found configuration
errors; `partial` means some could not run. An invalid invocation/root yields
`error`. Warnings are not code-level architecture findings.

| Exit | Meaning |
| --- | --- |
| 0 | Complete with passed/warning/not-applicable checks; reasons stay explicit |
| 1 | Complete with at least one configuration error |
| 2 | Invocation/root error |
| 3 | Partial or blocked dependent checks, even if errors were also found |
| 130 | Cancelled; never a clean result |

Recognized JSON mode emits one object on stdout, stderr for progress. Flag parse
failures before recognizing JSON mode may use stderr only. Human output and a
future MCP adapter consume the same internal typed result. MCP stays read-only,
confined to its configured project and uses structured output; no client-supplied
root override. No MCP tool or command is added by this proposal.

## Implementation and future verification

1. Establish internal diagnostic types and injectable readers; share safe path,
   schema and ownership validation with existing workspace/rule operations.
2. Implement version/root/index checks, then bundles/origins and rule/index checks.
3. Add CLI output and JSON contract tests. Consider MCP exposure only after the
   local implementation is stable; do not add background health checks.

Required fixtures: unconfigured directory; valid local customization; missing
versus malformed root index; duplicate scopes; invalid child under valid parent;
escaping local root; missing manifest/required file; optional file absent;
invalid/unknown origin; foreign rule filename collision; external index bytes;
incompatible Cursor frontmatter; stale managed reference; legacy scope index;
permission denial; symlink/junction in an ancestor or bundle; path-swap race;
malicious text; secrets never read/printed; cancellation and budgets. Assert exit
codes, stdout isolation, ordering, zero network activity and unchanged project
hashes on Windows/Linux. No test may repair fixtures to make diagnostics pass.

Planning-only schema examples and rejection checks run with
`pwsh -File docs/proposals/verify-contracts.ps1`; they do not implement these
runtime checks. Public YAML contracts remain unchanged.
