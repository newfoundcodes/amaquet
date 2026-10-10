---
title: "Glossary"
---

**AOF** — Append-only file containing journaled Amaquet mutation requests.

**DataType** — Stable numeric/wire identity for a Amaquet value.

**Encoding** — Internal representation of a value; conceptually separate from DataType.

**Amaquet** — Amaquet's framed network protocol.

**`amaquet://`** — Plain-TCP Amaquet endpoint URI.

**`amaquets://`** — TLS Amaquet endpoint URI.

**Typed wire value** — JSON object with `type` and `value` used when nesting values inside composite structures.

**Shard** — One of 256 independently locked key maps in the current engine.

**Version** — Monotonic per-key revision counter.

**Bootstrap token** — Administrative token configured or generated at startup.
