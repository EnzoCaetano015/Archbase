# Archbase

Archbase is an open-source CLI that gives AI coding agents explicit, reusable structural patterns. Patterns describe **how a type of code is written**; architecture rules describe **where code belongs and which relationships are allowed**.

This repository contains the completed core milestone and the initial canonical rules work (TASK-001 through TASK-015):

- the `arc` Go CLI with `help` and `version`;
- versioned YAML contracts backed by JSON Schema;
- safe filesystem primitives;
- a validated bundle loader for required and optional pattern files;
- seven structural Next.js and .NET patterns;
- ordered resolution across embedded, directory, and public Git registries;
- a concurrency-safe Git cache with a 15-minute TTL and validated stale fallback;
- transactional installation and creation of customizable local patterns;
- nearest-scope resolution for files, directories, and future paths;
- deterministic pattern resolution and inspection commands;
- an agent-neutral architecture rule contract and validated rule registry;
- initial modular Next and layered .NET architecture rules.

Rules exporters, rules CLI commands, and MCP are intentionally not implemented yet.

## Requirements

- Go 1.26 or newer

## Build and test

```bash
go test ./...
go vet ./...
go build -trimpath ./cmd/arc
```

Inject a release version with:

```bash
go build -ldflags "-X github.com/EnzoCaetano015/Archbase/internal/version.Value=0.1.0" ./cmd/arc
```

## CLI

```bash
arc help
arc version
arc add next/page@1234 ./src/pages
arc create pages-standard ./src/pages --from next/page@1234
arc resolve ./src/pages/Example.tsx
arc inspect next/page@1234
```

The official registry is embedded in the binary and therefore works offline. A public Git registry can be placed before it with `--registry-url`, `--registry-ref`, `--registry-subdir`, `--registry-cache-dir`, and `--registry-ttl`. Git access supports public `https`, `git`, and absolute `file` URLs without depending on a system Git executable.

Installed scopes use `.archbase/scope.yaml` and keep local pattern copies under `.archbase/patterns/`. Adding or creating another pattern preserves previous directories and atomically activates the new pattern. Existing pattern directories are never overwritten.

See [docs/schemas.md](docs/schemas.md) for the public YAML contracts, [docs/registry.md](docs/registry.md) for registry behavior, and [docs/rules.md](docs/rules.md) for the canonical rule model.
