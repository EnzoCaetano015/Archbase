# Interactive terminal UI

Run `arc` from your project directory to open the full-screen terminal application. Both standard input and output must be terminals. The dark interface uses a silver wordmark, turquoise selection, and readable sections; no special font or Nerd Font is required.

## Browse and act

The home screen shows the current project, installed version, and available actions. **Patterns** and **Rules** open searchable catalogs. Search matches IDs, names, and descriptions. Select an item to read its full definition, including source, version, files, scopes, restrictions, and rules.

Details wrap to the terminal width and scroll vertically. Long IDs and paths remain available in the detail view. Resize the terminal without losing your selection or form values. Narrow windows use a compact wordmark and a single column.

From a pattern's detail view, press `a` to install it. From a rule's detail view, press `e` to export it. The command palette also offers installation, creation, local inspection, scope resolution, rule removal, session settings, and MCP connection instructions.

## Keyboard shortcuts

| Context | Keys | Action |
| --- | --- | --- |
| Everywhere | `Ctrl+P` | Open or close the searchable command palette. |
| Navigation | Up / Down, `Enter` | Select and open an item or action. |
| Catalog | `/` | Focus the search field. |
| Catalog | `Ctrl+R`, `w` | Refresh the catalog or read registry warnings. |
| Details and messages | Up / Down, Page Up / Page Down | Scroll through all content. |
| Forms | `Tab`, `Shift+Tab` | Move between fields and the submit button. |
| Forms | `Enter`, Space, Left / Right | Advance text fields, toggle checkboxes, or change choices. |
| Forms | `Ctrl+S` | Validate and review the form. |
| Review | `Enter` | Apply the displayed operation. |
| Navigation | `Esc` | Close the palette, leave search, edit a review, or return. |
| Home | `q` | Return to the shell. |
| Everywhere | `Ctrl+C` | Cancel an active query; otherwise exit. |

The footer shows the shortcuts for the current screen. `Ctrl+C` during a confirmed write requests exit after that operation finishes or rolls back. New actions are disabled while a write is in progress.

## Filesystem operations

Installation, creation, export, and removal show a review screen before changing files. Read the ID, resolved destination, export format, and options, then press `Enter` to apply or `Esc` to edit. Review screens can be scrolled when paths or summaries are long.

Choose agents, claude, cursor, or copilot. Rules share a central document and every activated native index stays synchronized; unrelated agent files are not activated automatically. Overwrite is disabled by default and authorizes replacement only of a changed owned document for the same RuleID. Managed indices update automatically, so there is no merge checkbox. Rule removal accepts one RuleID or an explicit **ALL managed rule exports** selection; it preserves external instructions, native entry files, patterns, and scopes.

Failures open a scrollable message. Press `Esc` to return to the form with your values preserved. Existing scope precedence, validation, local customization, conflict protection, and rollback rules apply equally to the UI and shell commands.

## Project and registry

The initial project is the directory from which you launched `arc`. Form paths are resolved relative to the selected project. **Project & registry** lets you change that directory, public registry URL, ref, subdirectory, cache directory, and cache TTL. Initial registry settings come from the existing global flags:

```bash
arc --registry-url https://github.com/EnzoCaetano015/Archbase.git --registry-ref v0.3.0 --registry-subdir registry
```

Settings last only for the current session. They do not create a project configuration file or change your shell's working directory. Registry queries may update the existing registry cache. With no registry URL, the built-in catalog works offline.

## Shell, scripts, and MCP

Direct commands remain available with their existing arguments, flags, and exit codes:

```bash
arc rules list
arc rules inspect architecture/next-modular@1
arc --plain rules list
arc --plain
```

Interactive command output uses the same visual hierarchy as the application but returns directly to your shell. Output redirected to a file or pipe keeps the existing plain format. `--plain` disables both full-screen navigation and rich command formatting. `TERM=dumb` also selects the plain presentation. A non-empty `NO_COLOR` removes color without disabling the full-screen application.

`arc version` retains its original single-line output. `arc mcp serve` remains a separate, read-only stdio process: the UI never takes over its protocol streams or starts a server in the background. The **MCP connection** screen provides a client configuration for your selected project. See the [MCP guide](mcp.md).
The project directory shown on the home screen is the workspace root. All pattern installations share its single `.archbase/scopes.yaml`; changing the scope path does not create nested metadata directories. Session settings can change the root, and `arc --project-root <path>` selects it on startup. Edit the `.md` document at the pattern root printed by resolve/inspect, preserving its typed code fence.
