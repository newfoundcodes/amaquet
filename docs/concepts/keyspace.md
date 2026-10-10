---
title: "Keyspace and lifecycle"
---

This page explains how Amaquet stores keys, applies expiration, and performs conditional lifecycle operations.

## Keys

Keys are non-empty strings. There is one global in-process keyspace. The URI parser rejects non-empty path/database components because logical databases are not implemented.

## Common commands

Every currently registered logical type participates in the common lifecycle commands: `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, `DEL`, top-level `GET`, and cursor `SCAN`. `GET` returns either the direct value or a synchronized public snapshot, while `CAS` performs an optimistic versioned replacement.

## TTL behavior

`EXPIRE` accepts positive milliseconds. Reads enforce lazy expiry. The background expiration worker uses an indexed min-heap ordered by absolute expiry; it does not scan every key once per second. Each expiring key owns at most one scheduled heap node. Replacing the key, changing its TTL, or calling `PERSIST` updates or removes that node instead of accumulating stale expiry records.

`EXPIRE` and `PERSIST` first reject/purge a logically expired key, so an expired value cannot be revived by removing or changing its TTL.

`TTL` returns milliseconds remaining. `-1` means a live key has no expiry.

## `NX`, `XX`, and `CAS`

`SET` and `CREATE` accept `nx` and `xx`. `CAS` compares the current per-key version with `expected_version` and applies the new typed value only when they match.

## Enumeration

`KEYS` remains useful for bounded administration but collects its result by consuming shard-local sorted scan pages. The combined result is not globally lexical. For application iteration, use `SCAN`. Its opaque cursor encodes progress through shard-local sorted views, avoiding a complete keyspace materialization on every page.
