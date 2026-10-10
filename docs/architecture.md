---
title: "Amaquet architecture"
---

This page describes the server's storage engine, protocol boundary, persistence, and administration architecture.

## Storage engine

The keyspace is divided across 256 FNV-1a-selected shards. Each shard owns a map and an RW mutex. A stored entry contains:

- strongly typed `core.Value`;
- creation/update timestamps;
- optional expiry timestamp;
- monotonic per-key version;
- atomic access counter and last-access timestamp;
- approximate engine memory size used for memory governance.

The engine supports `SET` with NX/XX behavior, `GET`, deletion, TTL, persist, compare-and-set, in-place atomic updates, cursor scans, and an internal deterministic multi-key transaction primitive. Expiration is scheduled through a versioned min-heap and also enforced lazily on access. Engine key count and compression telemetry are maintained incrementally.

Complex values have their own fine-grained mutexes. Blocking queues, barriers, and semaphores do not hold a keyspace shard lock while waiting.

## Type and encoding separation

`core.DataType` is a stable public identifier. The concrete Go representation can evolve without changing the wire type name. This gives the project room to add packed/small-object encodings later while preserving protocol compatibility.

## Protocol server

The TCP server handles one frame-reader loop per connection and can execute multiple request IDs concurrently. Response writes are synchronized, and `CANCEL` can cancel an in-flight request context. It supports:

- optional TLS;
- API-key authentication;
- RBAC permission checks;
- command dispatch;
- Pub/Sub event frames;
- optional Ed25519 identity response in HELLO;
- a connection semaphore for the configured connection limit.

## Go client

`pkg/amaquet` has a background frame reader and a request-ID routing table. It supports concurrent requests on one connection and routes asynchronous subscription events to dedicated channels.

## Control plane

The HTTP control plane is separate from the Amaquet data protocol. It stores only management metadata in `data_dir/admin.json`:

- organization;
- team members;
- roles/RBAC;
- hashed API keys;
- audit events.

It never stores application/database keys in another database.

## Optional AOF

The current AOF is a transactional write-ahead journal (`AMQTAOF2`). A mutation is prepared before memory changes and is then committed or aborted. Recovery replays committed prepares only. Time-dependent requests are resolved to stable IDs/timestamps/absolute expirations before journaling, so restart time cannot change historical semantics.

AOF records have CRC protection. Background fsync errors become persistence-health failures. Legacy `AMQTAOF1` files are migrated when opened. Verified checkpoints and online compaction are available through the administration API.
