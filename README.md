# Archbase

Archbase is an open-source CLI that gives AI coding agents explicit, reusable structural patterns. Patterns describe **how a type of code is written**; architecture rules describe **where code belongs and which relationships are allowed**.

This repository currently contains the intermediate core milestone (TASK-001 through TASK-008):

- the `arc` Go CLI with `help` and `version`;
- versioned YAML contracts backed by JSON Schema;
- safe filesystem primitives;
- a validated bundle loader for required and optional pattern files;
- seven structural Next.js and .NET patterns;
- ordered resolution across embedded, directory, and public Git registries;
- a concurrency-safe Git cache with a 15-minute TTL and validated stale fallback.

Commands such as `arc add`, `arc create`, `arc resolve`, rules exporters, and MCP are intentionally not implemented yet.

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
```

The official registry is embedded in the binary and therefore works offline. The internal registry API can also read prepared directories or clone public `https`, `git`, and `file` Git sources without depending on a system Git executable. Git cache configuration is not exposed through CLI flags yet.

See [docs/schemas.md](docs/schemas.md) for the public YAML contracts and [docs/registry.md](docs/registry.md) for registry behavior.
