---
title: "Implementation coverage map"
---

This page maps the production repository to the documentation that describes it. It is an audit aid: a subsystem is not considered documented merely because its package name appears somewhere. Its public behavior, persistence boundary, security boundary, configuration, and operational constraints must have an owning page.

## Runtime components

Each production subsystem has an owning page for its externally observable behavior and operational boundary. Combined rows indicate implementation files that participate in one feature and are documented together.

| Source                                                            | Responsibility                                                                                                     | Documentation                                                                                                                               |
| ----------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `cmd/amaquet`                                                     | Process startup, configuration, listeners, AOF replay, bootstrap token, graceful shutdown                          | [Installation](../getting-started/installation.md), [Configuration](configuration.md), [Production deployment](../operations/deployment.md) |
| `internal/config`                                                 | Defaults, JSON loading/saving, environment overrides, validation, runtime config manager                           | [Configuration](configuration.md), [Getting-started configuration](../getting-started/configuration.md)                                     |
| `internal/core`                                                   | Sharded keyspace, entry metadata, TTL scheduler, versions, memory accounting, eviction, scans, staged transactions | [Engine internals](../concepts/engine-internals.md), [Keyspace](../concepts/keyspace.md), [Capacity](../operations/capacity.md)             |
| `internal/protocol`                                               | Fixed frame encoding/decoding, payload bounds, requests/responses, endpoint URI parsing                            | [Framing](../protocol/framing.md), [Connection URIs](../protocol/uri.md), [Wire values](../protocol/wire-values.md)                         |
| `internal/server/tcp.go`                                          | Amaquet connection lifecycle, TLS, request multiplexing, cancellation, authorization, limits, Pub/Sub events       | [Protocol overview](../protocol/overview.md), [Framing](../protocol/framing.md), [Authentication](../protocol/authentication.md)            |
| `internal/server/dispatcher.go`                                   | Commands, type operations, journaling boundary, deterministic replay arguments, chunked blobs                      | [Commands](../protocol/commands.md), [Type operations](../data-types/operations.md), [Persistence](../concepts/persistence.md)              |
| `internal/server/factory.go`                                      | Direct wire decoding and `CREATE` factories                                                                        | [Wire values](../protocol/wire-values.md), [Type operations](../data-types/operations.md)                                                   |
| `internal/compression` and `internal/server/value_compression.go` | Adaptive codecs, canonical encodings, transparent materialization                                                  | [Adaptive compression](../concepts/compression.md)                                                                                          |
| `internal/persistence`                                            | AMQTAOF1 migration, AMQTAOF2 transactions, replay, checkpoint, compaction, restore                                 | [Persistence](../concepts/persistence.md), [Backups](../operations/backups.md)                                                              |
| `internal/admin/state.go`                                         | Organization, members, credentials, TOTP, RBAC, API keys, audit, crash-safe state                                  | [Security](../concepts/security.md), [Admin API](admin-api.md)                                                                              |
| `internal/admin/http.go`                                          | Admin routes, sessions, metrics, and security headers                                                              | [Admin API](admin-api.md), [Observability](../operations/observability.md)                                                                  |
| `internal/security`                                               | Ed25519 identity generation, loading, signing, verification, fingerprinting                                        | [TLS and identity](../operations/tls.md)                                                                                                    |
| `internal/types`                                                  | The 91 registered value implementations and snapshots                                                              | [Data types](../data-types/index.md), [Type operations](../data-types/operations.md)                                                        |

## User-facing programs and libraries

The supported entrypoints are the public Go client and the four command-line programs. Container assets package the same server rather than introducing a separate runtime implementation.

| Source                         | User-facing surface                                                             | Documentation                                           |
| ------------------------------ | ------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `pkg/amaquet`                  | Concurrent Go client, TLS, protocol negotiation, commands, subscriptions, blobs | [Go client](go-client.md)                               |
| `cmd/amaquet-cli`              | Generic command-line client and JSON input modes                                | [CLI](cli.md)                                           |
| `cmd/amaquet-keygen`           | Ed25519 identity key generation                                                 | [CLI](cli.md), [TLS and identity](../operations/tls.md) |
| `cmd/amaquet-restore`          | Verified offline checkpoint installation                                        | [CLI](cli.md), [Backups](../operations/backups.md)      |
| `Dockerfile` and Compose files | Container build, local deployment, TLS production example                       | [Docker](../operations/docker.md)                       |

## Data-structure implementation files

The type catalog is organized by implementation family. Each family page summarizes the structures, while the operation reference and individual type pages define the wire contract.

| Source file               | Implemented families                                                           | Detailed pages                                                                            |
| ------------------------- | ------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| `scalars.go`              | strings, integers, exact decimals, UUID, symbols, time values                  | [Scalars](../data-types/scalar.md)                                                        |
| `collections.go`          | lists, arrays, deque/ring, sets, maps, tuples                                  | [Collections](../data-types/collections.md)                                               |
| `structured.go`           | JSON, opaque encodings, records, matrices, tensors                             | [Structured values](../data-types/structured.md)                                          |
| `queues.go`, `streams.go` | queues, streams/groups, Pub/Sub, retained topics, event logs                   | [Queues and messaging](../data-types/queues-and-messaging.md)                             |
| `bits.go`                 | bitmaps, bit fields, adaptive Roaring containers                               | [Bit structures](../data-types/bit-structures.md)                                         |
| `probabilistic.go`        | HLL, Bloom/Cuckoo filters, CMS, Top-K, t-digest                                | [Probabilistic structures](../data-types/probabilistic.md)                                |
| `timeseries.go`           | time series, counter/gauge series, histograms, aggregation                     | [Time series](../data-types/time-series.md)                                               |
| `geo.go`                  | points, boxes, polygons, grid-backed spatial index                             | [Geospatial](../data-types/geospatial.md)                                                 |
| `vectors.go`              | vector metrics, exact search, quantization, HNSW                               | [Vectors](../data-types/vectors.md)                                                       |
| `indexes.go`              | B-tree, hash/radix/trie/inverted/secondary indexes, document fields, adjacency | [Indexes and search](../data-types/indexes-and-search.md)                                 |
| `sync.go`, `network.go`   | counters, leases, locks, barriers, semaphores, token buckets, CIDR, URL        | [Graph, synchronization, and network](../data-types/graph-synchronization-and-network.md) |

## Deliberate boundaries

These exclusions are part of the current product contract. They prevent deployment or client documentation from implying distributed, relational, or Redis-compatible behavior that the implementation does not provide.

- Amaquet is single-node and has one logical keyspace. It does not implement clustering, replication, logical databases, SQL, Redis RESP compatibility, or an automatic query planner.
- Index values are explicit user-managed objects. Writing a document does not automatically update a separate index.
- AOF recovery covers journaled database mutations. HTTP cookie sessions, live connections, Pub/Sub subscribers, and synchronization ownership are process state and are not restored.
- `admin.json` is the control-plane store. It is not part of the user keyspace or the AOF.
- Engine memory accounting is approximate stored-data accounting, not an RSS hard limit.

## Automated documentation invariants

`python3 docs/scripts/check_docs.py` checks the documentation against source-derived inventories. It verifies all registered types, catalog operation counts, test/fuzzer pages, top-level commands, protocol opcodes, configuration fields, environment overrides, admin routes, public Go client methods, sidebar entries, frontmatter, explanatory prose for every section, and local links. The Astro production build additionally checks every generated internal link and asset.
