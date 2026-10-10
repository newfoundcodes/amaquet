---
title: "Persistence and recovery"
---

Amaquet keeps the active data set in its own in-memory engine. Optional persistence uses the Amaquet append-only write-ahead journal (AOF); it does not depend on Redis or another database.

## Transactional AOF v2

Current journals begin with:

```text
AMQTAOF2\n
```

After the header, each physical record is a 4-byte big-endian payload length, a 4-byte big-endian IEEE CRC-32 of the payload, and the JSON payload itself. A record payload is limited to 64 MiB and contains the transaction kind, transaction ID, resolved request when applicable, and record timestamp. A logical mutation uses these phases:

1. `prepare`: persist the fully resolved mutation before changing memory.
2. Apply the mutation to the in-memory engine.
3. `commit`: mark the prepare as committed.
4. `abort`: record that a prepared mutation failed before commit.

Recovery replays only prepares that have a matching commit. An uncommitted prepare is ignored. If the journal commit fails after memory has changed, Amaquet enters a persistence-degraded fail-stop state and rejects further durable mutations until the persistence problem is resolved or the process is restarted from valid durable state.

Legacy `AMQTAOF1` files are detected and migrated to the v2 journal format when they are opened.
Journals created before the Amaquet rename are also recognized by their historical v1/v2 signatures and rewritten with the current `AMQTAOF2` header before use.

## Deterministic replay

Journal records resolve time-dependent values before the mutation is written. Examples include:

- absolute key expiration timestamps;
- `CAS` expiration timestamps;
- automatically generated stream IDs;
- time-series sample timestamps;
- persistent-topic and event-log timestamps;
- reliable-queue enqueue and claim times;
- delayed-queue readiness checks.

Replay reconstructs the historical mutation instead of generating a new value from the restart clock. If an absolute expiration has already passed during recovery, the key remains absent.

## Integrity and failure state

Every record is protected by CRC-32. A partial final record can be ignored as an interrupted tail write. A checksum failure in a complete record is corruption and stops normal replay.

For `fsync: everysec`, background flush errors are retained as persistence-health failures. Readiness reports the degraded state, and new durable mutations are rejected instead of silently running without the requested durability.

## Durability modes

`persistence.fsync` controls when buffered journal phases are forced toward durable storage. The modes trade write latency for the amount of recently acknowledged data that can be lost after an operating-system or machine failure.

- `always`: flush and fsync every journal phase. Highest durability and highest write latency.
- `everysec`: flush approximately once per second. This is the normal balance for many deployments.
- `no`: use buffered and operating-system writeback. Lowest durability.

## Compact verified checkpoints

`POST /api/persistence/checkpoint` creates a compact recovery image rather than copying the live journal byte-for-byte. Amaquet:

1. serializes with AOF writers and flushes the active journal to capture a committed prefix;
2. reads that stable prefix while later appends wait;
3. resolves committed requests;
4. removes aborted and uncommitted records;
5. squashes obsolete histories where command semantics permit it and retains only the latest completed blob-upload sequence for each blob key;
6. writes a fresh `AMQTAOF2` image to a temporary file;
7. fsyncs and replays the temporary image for verification;
8. atomically publishes the checkpoint and fsyncs the destination directory.

Incomplete chunked uploads are not restored as live uploads.

## Online compaction

Online AOF compaction uses the same verified rewrite principles. The replacement uses a temporary file, verification, backup/rollback protection, atomic rename, and directory fsync. It removes aborted/uncommitted transactions and collapses supported superseded histories while preserving observable committed state.

Compaction is not a substitute for backup. Keep independent checkpoint copies on separate storage.

## Offline restore

Use `amaquet-restore` while the target Amaquet process is stopped:

```bash
amaquet-restore \
  -source /backups/amaquet-checkpoint.aof \
  -target /var/lib/amaquet/appendonly.aof
```

The restore command validates the source, copies it to a temporary target, fsyncs it, verifies it again, then atomically installs it with rollback protection.

## Runtime state that is not recovered

Ephemeral process coordination is intentionally not restored when doing so could recreate stale ownership. Examples include live Pub/Sub subscriptions, HTTP cookie sessions, active protocol connections, and process-local synchronization ownership.

See [Backup and recovery](../operations/backups.md) for operational procedures.
