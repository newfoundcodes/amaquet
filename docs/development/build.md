---
title: "Development build"
---

Use these commands and Make targets to build, test, and serve Amaquet and its documentation during development.

## Go

Run the unit/integration suite, race detector, vet, and builds separately so a failure identifies the affected layer. The build command covers all four shipped executables.

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/amaquet ./cmd/amaquet-cli ./cmd/amaquet-keygen ./cmd/amaquet-restore
```

## Documentation

Use these targets to build the production documentation site or run its local development server.

```bash
make docs-build
make docs-serve
```

These targets install the locked Node dependencies under `docs/node_modules` and invoke Astro from that package. `make docs-check` remains dependency-free.

## Make targets

The root Makefile includes normal build/test targets plus Bash test, fuzz-smoke, docs-check, docs-build, and docs-serve targets.
