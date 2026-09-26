# First Archbase flow with Next

This guide starts from an empty project directory and demonstrates a remote pattern, a customizable local pattern, hierarchical scope resolution, architecture rules, and the MCP server. Install `arc` first by following [installation.md](installation.md).

The commands pin the public Archbase registry to `v0.1.0`, so the result does not change when `main` advances. No global configuration file is created.

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
--registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry
```

## 2. Install the remote page pattern

Install `next/page@1234` at the project root:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry add next/page@1234 .
```

The command creates `.archbase/scope.yaml` and stores the validated bundle under `.archbase/patterns/page-1234`. The scope records the Git source, original PatternID, and version.

Inspect the installed scope and the registry pattern:

```bash
arc resolve src/pages/Home.tsx
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry inspect next/page@1234
```

The target file does not need to exist. Resolution walks its ancestors and selects the project scope.

## 3. Derive and customize a local pattern

Create a nested scope derived from the same remote bundle:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry create pages-standard src/pages/admin --from next/page@1234
```

The active ID in that scope is `local/pages-standard@1`. Its files are stored at:

```text
src/pages/admin/.archbase/patterns/pages-standard/Example/Example.tsx
src/pages/admin/.archbase/patterns/pages-standard/Example/Example.hook.ts
src/pages/admin/.archbase/patterns/pages-standard/Example/Example.utils.ts
```

Edit `Example/Example.tsx` inside that local pattern directory and add a recognizable line such as `// Local admin customization`. Archbase preserves those bytes; it does not replace local customization with registry content.

Now compare the two resolutions:

```bash
arc resolve src/pages/Home.tsx
arc resolve src/pages/admin/Dashboard.tsx
arc inspect src/pages/admin/Dashboard.tsx
```

The first target resolves to `next/page@1234` in the project scope. The nested target resolves to `local/pages-standard@1`, while its origin remains `next/page@1234`. This is the nearest-scope-wins rule.

## 4. Export the Next architecture rule

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry rules inspect architecture/next-modular@1
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry rules add architecture/next-modular@1 --format cursor --destination .
```

The export creates `.cursor/rules/architecture-next-modular-1.mdc`. It references the four Next PatternIDs and their scopes without copying pattern source examples.

Copilot and hierarchical AGENTS exports use the same canonical rule:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry rules add architecture/next-modular@1 --format copilot --destination .
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.1.0 --registry-subdir registry rules add architecture/next-modular@1 --format agents --destination .
```

If an `AGENTS.md` already exists, repeat the AGENTS command with `--merge`. Cursor and Copilot require `--overwrite` when intentionally replacing their generated files.

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
        "v0.1.0",
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
| `get_pattern_files` | `{"path":"src/pages/admin/Dashboard.tsx"}` | Returns the edited local `Example.tsx` content. |
| `get_scope_rules` | `{"path":"src/pages/admin/Dashboard.tsx"}` | Matches `architecture/next-modular@1` through the local pattern's origin. |
| `list_project_scopes` | `{}` | Lists `.` and `src/pages/admin` in deterministic order. |

For the complete tool contracts, confinement rules, text/base64 behavior, and error semantics, see [mcp.md](mcp.md).
