# Next.js Architecture Auditor

Next.js Auditor is the Premium Archbase workflow for evidence-based architecture
reviews. It inspects App Router boundaries, imports and layer responsibilities,
then produces a Markdown report with confirmed findings, suspicions, unassessed
checks, severities, RuleIDs and prioritized recommendations.

## Availability

The initial workflow is implemented. **The paid Agensi listing is not published
yet; purchasing is not available from this page.** Listing approval, receiving
payments and commercial terms are separate launch steps. This page makes no
promise about a publication date or final purchase price.

## What stays free

The Go CLI, read-only MCP server, official rules, patterns, core fixes and
[Architecture Starter](skills.md) remain public and MIT licensed. Premium adds
original workflows, report templates and guidance. Its original materials have
a proprietary license; included public catalog references retain MIT attribution.
CLI installation and documentation remain openly accessible.

## Example request and limits

```text
Use nextjs-auditor for ./apps/web. Review the App Router and relevant imports.
Return an evidence-based architecture report. Do not change files.
```

The agent must first load the skill through an authorized delivery channel.
The workflow can use included references without `arc`; an existing trusted CLI
or confirmed-project MCP connection is optional. It respects local conventions
and requires real code evidence before confirming a violation. Unresolved imports
and incomplete context remain explicit limitations.

This is an agent-guided source review, not an AST scanner, runtime verifier or
complete security audit. It does not implement fixes automatically. The skill
adds no telemetry; your chosen agent/provider still processes supplied context
under its own configuration. Do not supply secrets.

Version 0.2.0 was evaluated in 30 independent Codex CLI supplied-snapshot sessions,
with semantic review and unchanged fixture hashes. Native discovery and behavior
in Cursor, Claude Code and Copilot have not been verified. Documented packaging
conformance does not certify those hosts. Agensi loading is not yet live-tested.

For initial orientation today, use the [free Starter](skills.md). The
[public Skills workflow](https://github.com/EnzoCaetano015/Archbase/actions/workflows/skills.yml)
provides generated Starter archives and checksums; GitHub artifact downloads may
require sign-in. Premium launch links will be added here after approval.
