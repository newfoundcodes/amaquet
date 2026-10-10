---
title: "Backup and recovery"
---

This page explains which Amaquet state must be protected and how to create, validate, and restore recovery artifacts.

## What to protect

Protect these files according to your security and retention policy:

- the active AOF;
- verified checkpoint files;
- `admin.json`;
- the JSON configuration;
- TLS private keys and certificates;
- optional Ed25519 identity keys;
- the protected bootstrap-token file while bootstrap is still active.

Member access hashes and TOTP secrets are part of `admin.json`. Treat an administration-state backup as sensitive security material.

## Create an online checkpoint

Use `POST /api/persistence/checkpoint`. The AOF serializes the checkpoint with journal writers, flushes the file, and captures a committed prefix while later appends wait. Unlike online compaction, the checkpoint callback is not wrapped in the dispatcher's broader engine mutation barrier.

The checkpoint process:

1. flushes the active journal;
2. resolves committed AOF transactions;
3. compacts semantics-preserving obsolete history;
4. writes a fresh AMQTAOF2 recovery image;
5. fsyncs the output;
6. replays the temporary result to validate framing, CRCs, transaction structure, and request decoding;
7. atomically publishes the checkpoint.

The checkpoint is therefore a compact recovery image, not a blind copy of a file that is changing underneath the backup process.

## Compact the active AOF

`POST /api/persistence/compact` rewrites the active journal online. The compactor removes aborted and uncommitted records and collapses overwritten state where this is safe.

For chunked blobs, only the latest completed upload sequence for the key is retained. Incomplete upload sequences are not recovery state.

## Restore offline

Stop Amaquet before replacing its active AOF.

```bash
amaquet-restore \
  -source /backups/checkpoint-20261002T120000Z.aof \
  -target /var/lib/amaquet/amaquet.aof
```

The restore tool:

1. validates the source by replay parsing;
2. copies it to a temporary target;
3. fsyncs the temporary file;
4. validates the copied result;
5. moves the old target to a temporary rollback file when present;
6. atomically installs the restored AOF;
7. fsyncs the destination directory;
8. removes the rollback file after success.

Do not run `amaquet-restore` against an AOF that an active Amaquet process has open.

## Recovery drill

A production backup is not complete until it has been restored in a separate environment. Periodically test:

- checkpoint creation;
- `amaquet-restore` into a clean data directory;
- server startup and AOF replay;
- key counts and application-level invariants;
- administration state and RBAC recovery;
- TLS/identity key availability.

Preserve a corrupt source file before manual repair. A checksum mismatch in a complete record is treated as corruption and is not silently skipped.
