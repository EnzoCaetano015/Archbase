<p align="center">
  <img src="./docs/img/logo_without_background.png" alt="Archbase — Architecture and Code Standards">
</p>

<h1 align="center">Archbase</h1>

<p align="center">
  <strong>Architecture that AI coding agents can actually follow.</strong>
</p>

Archbase is an open-source Go CLI (`arc`) for sharing code patterns and architecture rules with developers and AI coding agents.

- **Patterns** define how a type of code is structured.
- **Rules** define paths, responsibilities, and dependency constraints.
- **Scopes** select the active pattern for each part of a project.

The built-in catalog works offline. Local patterns can be customized, rules can be exported to Cursor, GitHub Copilot, and `AGENTS.md`, and compatible agents can inspect project context through read-only MCP tools.

## Quick start

Install on Linux (official distribution):

```bash
curl -fsSL https://archbase.caetanodev.com/install.sh | sh
```

Open a new terminal, then run these commands from your project directory:

```bash
arc version
arc add next/page@1234 .
arc resolve src/pages/Home.tsx
arc inspect next/page@1234
```

`arc add` stores the pattern in a local `.archbase` scope. `arc resolve` finds the active pattern even when the target file does not exist yet. Use `arc help` for available commands.

macOS and Windows are available as **unsigned previews**. See the [installation guide](docs/installation.md) for requirements, manual installation, and removal.

## Documentation

| Guide | Contents |
| --- | --- |
| [Installation](docs/installation.md) | Installers, supported platforms, checksums, and uninstall. |
| [Getting started](docs/getting-started.md) | Complete workflow with patterns, customization, rules, and MCP. |
| [Concepts](docs/concepts.md) | Patterns, nested scopes, resolution, and local customization. |
| [Registry](docs/registry.md) | Pattern catalog, public Git sources, caching, and validation. |
| [Architecture rules](docs/rules.md) | Rule catalog and exports for Cursor, Copilot, and `AGENTS.md`. |
| [MCP](docs/mcp.md) | Read-only tools, client configuration, and project boundaries. |
| [Schemas](docs/schemas.md) | Public YAML contracts and versioning. |
| [Development](docs/development.md) | Build, validation, contributions, and release workflow. |
| [Code signing](docs/code-signing-policy.md) | Current signing status and release integrity policy. |
| [Privacy](docs/privacy.md) | Network access and local data. |

## License and maintainer

[MIT License](LICENSE). Maintained by [Enzo Caetano](https://github.com/EnzoCaetano015). Archbase does not collect telemetry or personal data.

Download published versions from [GitHub Releases](https://github.com/EnzoCaetano015/Archbase/releases/latest).
