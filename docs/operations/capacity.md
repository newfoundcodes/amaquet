---
title: "Capacity planning"
---

Amaquet keeps the active database in memory. Plan capacity from the stored data, indexes, Go object overhead, protocol buffers, temporary work space, persistence buffers, connections, and operating-system headroom.

## Memory budget and process guard

Set `limits.max_memory_bytes` for the engine data budget. Choose `noeviction` when losing an arbitrary key is unacceptable, or a supported eviction policy when bounded cache behavior is desired.

The engine tracks an approximate stored-data budget. It also exposes Go heap statistics and sets a Go runtime soft memory limit with additional headroom when a Amaquet memory limit is configured. This is not an exact RSS limit. TLS, the runtime, page cache, file mappings, stacks, temporary buffers, and the operating system can make process RSS differ from the configured data budget.

Do not set the Amaquet data budget equal to machine RAM. Keep explicit headroom and enforce an operating-system/container memory limit as the final safety boundary.

## Expiration

TTL expiration uses lazy expiry plus an indexed min-heap. There is at most one scheduled heap node for the current expiration state of a key, so repeated `EXPIRE`/`PERSIST` cycles do not accumulate stale expiration records indefinitely.

## Eviction

LRU/LFU/TTL eviction uses bounded candidate sampling instead of scanning the full keyspace for every victim. Sampling keeps eviction work bounded, but it is approximate by design.

## Key iteration

Use cursor `SCAN` for large databases. `KEYS` is convenient for bounded administrative use, but it still collects every matching key. Both commands sort within each shard and append shards in shard-index order; neither promises one global lexical ordering. `SCAN` returns an opaque continuation cursor without materializing the complete database in one result.

## Large binary values

A physical Amaquet frame remains limited to 64 MiB. Chunked upload lets a final binary value reach the hard 1 GiB per-object limit while each request remains bounded; `limits.max_pending_upload_bytes` separately controls the aggregate reservation for active uploads.

Upload behavior is deliberately different from the final in-memory representation:

- incoming chunks are spooled to bounded temporary files instead of preallocating the declared object size in the Go heap;
- zero-length chunks are rejected;
- overlapping chunks are rejected;
- the number of spans/chunks is capped;
- `connection_scoped: true` binds an upload to the exact native-protocol connection; the default upload is not connection-scoped;
- abandoned uploads expire, and connection-scoped uploads are also removed when their owner connection closes;
- pending upload bytes have a global reservation limit;
- incomplete uploads found during recovery are discarded;
- final large values use bounded `ChunkedBinary` blocks rather than one contiguous object-sized byte slice;
- clients read large values with `BLOB_READ` ranges instead of forcing one large response allocation.

For sustained large-object workloads, also budget the data directory for upload spool files and persistence checkpoints.

## Network and command concurrency

Amaquet has per-connection and global in-flight request limits plus a global inbound-payload budget. These limits protect the process from a client declaring many maximum-size frames at once. Configure them according to available RAM and expected concurrency.

`limits.command_timeout_ms` bounds ordinary command execution. Blocking primitives remain bounded by the server command context and their own requested timeout.

## Compression

Adaptive compression can reduce stored payload bytes for repetitive text, JSON, zero-filled numeric data, and similar values. Compression does not reduce every structure: active indexes, synchronization structures, graph links, key strings, and runtime metadata can remain native.

Measure both logical and stored bytes. Very small or incompressible values normally remain uncompressed.

## Indexes and algorithms

Production-oriented implementations include multi-level HNSW with bounded neighbor graphs, grid-backed geospatial candidate lookup, B-tree traversal for ranges, hybrid Roaring containers, bounded-memory Top-K candidates, and one-pass t-digest compression. Benchmark these structures with your real dimensions, cardinalities, and query distributions before setting production limits.
