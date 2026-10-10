---
title: "Repository layout"
---

This diagram maps the repository's executable, implementation, client, automation, test, and documentation directories.

```mermaid
flowchart TB
  Repository["Amaquet repository"]

  Repository --> Cmd["cmd/"]
  Cmd --> Server["amaquet/"]
  Cmd --> CLI["amaquet-cli/"]
  Cmd --> Keygen["amaquet-keygen/"]
  Cmd --> Restore["amaquet-restore/"]

  Repository --> Internal["internal/"]
  Internal --> Admin["admin/"]
  Internal --> Compression["compression/"]
  Internal --> Config["config/"]
  Internal --> Core["core/"]
  Internal --> Persistence["persistence/"]
  Internal --> Protocol["protocol/"]
  Internal --> Security["security/"]
  Internal --> TCPServer["server/"]
  Internal --> Types["types/"]

  Repository --> Package["pkg/amaquet/"]
  Repository --> Scripts["scripts/"]
  Repository --> Tests["tests/"]
  Repository --> Docs["docs/"]
  Docs --> AstroConfig["astro.config.mjs"]
```

The `cmd` directory contains the server, CLI, identity generator, and offline restore executables. `internal` contains the HTTP administration API and state, compression, configuration, sharded core engine, append-only persistence, protocol framing, identity, TCP dispatch, and concrete data structures. `pkg/amaquet` is the public Go client; `scripts`, `tests`, and `docs` contain project automation, integration coverage, and the Astro/Starlight documentation site respectively.

The `internal` boundary prevents external Go packages from depending directly on implementation details. Public applications should use `pkg/amaquet` or implement the Amaquet protocol.
