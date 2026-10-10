---
title: "Memory and concurrency model"
---

This page explains entry metadata, shard-level concurrency, object-level synchronization, and transaction locking.

## Key entries

Each key maps to an `Entry` containing:

- strongly typed `Value`;
- creation and update timestamps;
- optional expiry timestamp;
- monotonically increasing version;
- atomic access counter.

## Sharding

Amaquet currently creates 256 shards. The shard is selected with FNV-1a over the complete key. Independent shards can be accessed concurrently.

## Versions

A new key starts at version 1. Replacing a key, changing TTL, removing TTL, scalar mutation, or successful in-place type mutation advances its version. `CompareAndSet` can replace a value only if the observed version still matches.

## Atomic multi-key transactions

`Engine.Transaction` computes the distinct shards for all involved keys, sorts shard indexes, locks them in stable order, builds a view of non-expired entries, and runs the callback while those shard locks remain held. Stable lock ordering prevents lock-order inversion between transactions.

## Composite object locking

Many data types have their own `sync.RWMutex` or synchronization primitive. Their operation executes without a shard lock. After success, the engine takes the short shard lock needed to update the entry version and update timestamp.

## Blocking types

Blocking queues, barriers, and semaphores must not hold keyspace shard locks while waiting. The dispatcher retrieves the object, performs the wait on the object itself, then updates key metadata after the operation completes.

## Compressed physical encodings

A key's logical `DataType` and its physical storage encoding are separate. Large eligible values can be represented by a compressed payload while the entry retains the same public type ID. Reads materialize the native value only for the operation. See [Adaptive in-memory compression](compression.md).
