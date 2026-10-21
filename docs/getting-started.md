# First Archbase flow with Next

This guide starts from an empty project directory and demonstrates a remote pattern, a customizable local pattern, hierarchical scope resolution, architecture rules, and the MCP server. Install `arc` first with the one-command installer in [installation.md](installation.md).

For an interactive version of these workflows, run `arc` from your project and use the catalogs and forms. `Ctrl+P` opens the action palette. The [terminal UI guide](terminal-ui.md) covers navigation and session registry settings; the commands below remain available directly in your shell.

This guide targets the v0.5.0 release; until that tag is published, use the built-in catalog by omitting the registry flags. Run every command from the same project root.

The commands pin the public Archbase registry to `v0.5.0`, so the result does not change when `main` advances. No global configuration file is created.

## 1. Create a minimal project

```bash
mkdir archbase-next-demo
cd archbase-next-demo
mkdir -p src/pages/admin
```

In PowerShell, replace the last command with:

```powershell
New-Item -ItemType Directory -Force src/pages/admin | Out-Null
```

All registry-backed commands below use these global options:

```text
--registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry
```

## 2. Install the remote page pattern

Install `next/page@1234` at the project root:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry add next/page@1234 .
```

The command creates the root `.archbase/scopes.yaml` and stores the validated bundle under `.archbase/patterns/_root/next--page--1234`. The scope records the Git source, original PatternID, and version.

Inspect the installed scope and the registry pattern:

```bash
arc resolve src/pages/Home.tsx
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry inspect next/page@1234
```

The target file does not need to exist. Resolution reads the central index and selects the most specific directory scope.

## 3. Derive and customize a local pattern

Create a nested scope derived from the same remote bundle:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry create pages-standard src/pages/admin --from next/page@1234
```

The active ID in that scope is `local/pages-standard@1`. Run `arc resolve src/pages/admin/Dashboard.tsx` and use the printed pattern root to find these documents:

```text
<printed-pattern-root>/Example/Example.tsx.md
<printed-pattern-root>/Example/Example.hook.ts.md
<printed-pattern-root>/Example/Example.utils.ts.md
```

Edit the code inside the existing `tsx` fence of `Example/Example.tsx.md` and add a recognizable line such as `// Local admin customization`. Archbase preserves those bytes; it does not replace local customization with registry content.

Now compare the two resolutions:

```bash
arc resolve src/pages/Home.tsx
arc resolve src/pages/admin/Dashboard.tsx
arc inspect src/pages/admin/Dashboard.tsx
```

The first target resolves to `next/page@1234` in the project scope. The nested target resolves to `local/pages-standard@1`, while its origin remains `next/page@1234`. This is the nearest-scope-wins rule.

## 4. Export the Next architecture rule

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry rules inspect architecture/next-modular@1
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry rules add architecture/next-modular@1 --format cursor --destination .
```

The export creates `.archbase/rules/architecture-next-modular-1.md` and an always-applied index in `.cursor/rules/archbase.mdc`. The central document references the four Next PatternIDs without copying examples.

Activate other native entry points using the same shared document:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry rules add architecture/next-modular@1 --format copilot --destination .
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.5.0 --registry-subdir registry rules add architecture/next-modular@1 --format agents --destination .
```

Activate Claude with `arc rules add architecture/next-modular@1 --format claude --destination .`. AGENTS and Claude use root files; Copilot uses `.github/copilot-instructions.md`. All active indices stay synchronized. Existing framework instructions are preserved automatically; `--merge` is obsolete. A changed owned central document requires `--overwrite`; identical documents are reused. See [conflicts and Cursor frontmatter](rules.md).

## 5. Connect an MCP client

Resolve the absolute path of `archbase-next-demo`, then register this stdio server in an MCP client:

```json
{
  "mcpServers": {
    "archbase": {
      "command": "arc",
      "args": [
        "--registry-url",
        "https://github.com/EnzoCaetano015/Archbase.git",
        "--registry-ref",
        "v0.5.0",
        "--registry-subdir",
        "registry",
        "mcp",
        "serve",
        "--project-root",
        "/absolute/path/to/archbase-next-demo"
      ]
    }
  }
}
```

On Windows, use an absolute Windows path for `--project-root`. The client launches `arc`; do not start it in a separate terminal because stdout is reserved for MCP messages.

Call the six read-only tools with these representative arguments:

| Tool | Arguments | Expected observation |
| --- | --- | --- |
| `search_patterns` | `{"query":"page"}` | Includes `next/page@1234`. |
| `get_pattern` | `{"patternId":"next/page@1234"}` | Returns the complete validated manifest and Git source. |
| `resolve_pattern` | `{"path":"src/pages/admin/Dashboard.tsx"}` | Selects `local/pages-standard@1` and reports its remote origin. |
| `get_pattern_files` | `{"path":"src/pages/admin/Dashboard.tsx"}` | Returns extracted code from the edited local `Example.tsx.md`, with language `tsx` and no fences. |
| `get_scope_rules` | `{"path":"src/pages/admin/Dashboard.tsx"}` | Matches `architecture/next-modular@1` through the local pattern's origin. |
| `list_project_scopes` | `{}` | Lists `.` and `src/pages/admin` in deterministic order. |

For the complete tool contracts, confinement rules, text/base64 behavior, and error semantics, see [mcp.md](mcp.md).
