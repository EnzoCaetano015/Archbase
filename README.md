# Archbase

Archbase is an open-source CLI that gives AI coding agents explicit, reusable structural patterns. Patterns describe **how a type of code is written**; architecture rules describe **where code belongs and which relationships are allowed**.

This repository currently contains the foundation milestone (TASK-001 through TASK-004):

- the `arc` Go CLI with `help` and `version`;
- versioned YAML contracts backed by JSON Schema;
- safe filesystem primitives;
- an offline embedded registry with a provider for prepared Git checkouts.

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

The official foundation registry is embedded in the binary and therefore works offline. Git network access and cache management will be introduced with pattern resolution; this milestone can read an already-prepared checkout through its internal directory provider.

See [docs/schemas.md](docs/schemas.md) for the public YAML contracts.
