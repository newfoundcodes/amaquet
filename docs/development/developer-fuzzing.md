---
title: "Developer workflow fuzzing"
---

Amaquet has a second black-box fuzz layer in addition to the 91 per-type fuzzers. The per-type suite proves that each stable wire type survives invalid operations and still executes its normal semantic contract. The developer-workflow suite combines multiple features in the ways an application is more likely to use them.

No finite test suite can enumerate every possible program. This suite instead targets the main state transitions, concurrency patterns, failure boundaries, persistence paths, protocol behaviors, and administrative actions that a Amaquet client can exercise.

## Run the complete developer fuzz suite

This command combines bounded cross-feature scenarios, per-type mutations, and raw-frame fuzzing in one local run.

```bash
AMAQUET_FUZZ_SEED=4242 \
AMAQUET_SCENARIO_ITERATIONS=20 \
AMAQUET_FUZZ_ITERATIONS=25 \
AMAQUET_FRAME_FUZZ_ITERATIONS=500 \
./tests/bash/run_developer_fuzz.sh
```

The seed is printed by the scenario tests so a failure can be reproduced.

For a smaller scenario-only run:

```bash
AMAQUET_SCENARIO_ITERATIONS=5 \
AMAQUET_TEST_JOBS=2 \
./tests/bash/run_scenarios.sh
```

Security and persistence tests use isolated servers because they change authentication policy, restart the process, or deliberately damage persistence files:

```bash
./tests/bash/run_isolated_fuzz.sh
```

## Scenario coverage

The `tests/bash/scenarios` directory currently contains 28 fuzz/scenario files. They exercise:

| Area                     | Cases                                                                                                   |
| ------------------------ | ------------------------------------------------------------------------------------------------------- |
| Key lifecycle            | `SET`, `GET`, `EXISTS`, `TYPE`, `KEYS`, `MEMORY`, `DEL`, expiry and persistence of TTL                  |
| Conditional writes       | `NX`, `XX`, duplicate writes and missing-key updates                                                    |
| Key encoding             | spaces, separators, quotes, backslashes, long keys, emoji and non-Latin Unicode                         |
| Validation               | missing fields, invalid types, invalid base64, unsupported operations and wrong-type access             |
| Numbers                  | signed/unsigned 64-bit boundaries, large integers, high-precision decimal values and tiny finite floats |
| Binary/text              | randomized binary round trips and multilingual UTF-8 strings                                            |
| Collections              | list, deque, ring-buffer and set churn                                                                  |
| Maps/records             | repeated hash, ordered-map and multimap field mutations                                                 |
| Queue semantics          | FIFO, LIFO, priority, delayed and reliable queues                                                       |
| Reliable delivery        | `CLAIM`, `ACK`, `NACK`, retry count and dead-letter transition                                          |
| Blocking coordination    | a blocked queue consumer waking after enqueue and multi-client barrier release                          |
| Streams/events           | streams, consumer groups, persistent topics and event logs                                              |
| Time series              | out-of-order inserts, ranges, last value and aggregation                                                |
| Geo/vector               | geospatial radius queries, vector sets and HNSW search                                                  |
| Search/indexes           | B-tree, hash, radix, inverted and secondary-index workloads                                             |
| Probabilistic structures | Bloom, counting Bloom, Cuckoo, Count-Min Sketch, HyperLogLog, Top-K and t-digest invariants             |
| Compression              | raw-to-compressed-to-raw transitions and transparent read-back                                          |
| Large values             | bounded payloads from KiB through multi-MiB values using CLI `@file` input                              |
| Expiration               | batches of short-lived keys and keyspace cleanup                                                        |
| Concurrency              | atomic counter increments, concurrent map/set writes and connection churn                               |
| Mixed workloads          | randomized reads, writes, collection operations, metadata reads and key scans                           |
| Synchronization          | leases, locks, semaphores and token-bucket rate limits                                                  |
| Type replacement         | delete and recreate the same key under multiple unrelated data types                                    |
| Introspection            | `INFO` and `MEMORY` behavior while the keyspace changes                                                 |

## Protocol workflow fuzzing

The protocol directory contains raw-socket tests that bypass `amaquet-cli` where necessary. They cover:

- malformed Amaquet frames;
- random request IDs;
- many outstanding request IDs on one connection;
- frames delivered in one-to-seven-byte partial writes;
- declared payloads above the configured server limit;
- malformed clients followed by a clean reconnect;
- Pub/Sub subscribe, event delivery, and unsubscribe on separate client connections; and
- normal URI and command smoke tests after fuzz traffic.

A valid `PING` after malformed traffic is an important invariant. Protocol fuzzing is not considered successful merely because the malformed connection was closed; the server must continue to serve healthy clients.

## Security and admin API fuzzing

`tests/bash/security` starts a server with `protocol.require_auth=true`. It verifies:

- unauthenticated protocol requests are rejected;
- bootstrap authentication works;
- developer keys can mutate data;
- auditor keys can read but cannot write data;
- role restrictions also apply to the admin HTTP API;
- revoked credentials cannot open a new authenticated connection;
- expired API keys are rejected and future-expiring keys work;
- malformed JSON is rejected;
- unknown request fields are rejected where strict decoding applies;
- unsupported HTTP methods return the correct class of failure; and
- invalid member roles and unknown database commands do not damage service health.

## Persistence and restart fuzzing

`tests/bash/persistence` uses `fsync=always` for deterministic black-box persistence tests. It covers:

1. durable mutation replay after a clean restart;
2. preservation of lists and persistent topics, not only scalar values;
3. a truncated final AOF record, where earlier complete records remain recoverable;
4. deliberate checksum corruption, which must stop startup rather than silently accept damaged data;
5. replay of values that are adaptively compressed again in memory; and
6. organization/member/API-key control-plane state across restart.

Transient live coordination state such as active subscriptions and lock ownership is intentionally outside AOF durability. See [Persistence](../concepts/persistence.md).

## Native Go fuzzing

Amaquet also uses Go's coverage-guided fuzz engine. Seed corpora run as part of ordinary `go test ./...`.

Run every native target for a bounded period:

```bash
AMAQUET_GO_FUZZ_TIME=10s ./scripts/run_go_fuzz.sh
```

Current native targets cover:

- frame decoding with arbitrary byte input;
- frame write/read round trips;
- `amaquet://` and `amaquets://` URI parsing;
- compression round trips for RLE, LZ4, fast DEFLATE and adaptive mode;
- malformed compressed payload decoding;
- typed value decoding;
- nested wire-value decoding; and
- core engine key lifecycle with arbitrary keys and values.

All fuzz targets bound input sizes before allocating large buffers. This prevents a fuzz run from turning an accidental generated length into an uncontrolled resource-exhaustion test.

## Reproduction controls

Use these environment variables to make randomized runs repeatable and to adjust their breadth or duration.

| Variable                        | Purpose                                            | Typical value     |
| ------------------------------- | -------------------------------------------------- | ----------------- |
| `AMAQUET_FUZZ_SEED`             | Reproducible Bash scenario randomness              | `4242`            |
| `AMAQUET_SCENARIO_ITERATIONS`   | Iterations in cross-feature scenarios              | `10` to `100`     |
| `AMAQUET_FUZZ_ITERATIONS`       | Iterations in each per-type fuzzer                 | `25` to `1000`    |
| `AMAQUET_FRAME_FUZZ_ITERATIONS` | Random raw-frame cases                             | `100` to `10000`  |
| `AMAQUET_TEST_JOBS`             | Parallel scenario/type workers                     | `2` to `8`        |
| `AMAQUET_CONCURRENCY`           | Concurrent client workers in concurrency scenarios | `2` to `32`       |
| `AMAQUET_GO_FUZZ_TIME`          | Time per native Go fuzz target                     | `5s`, `30s`, `2m` |
| `AMAQUET_KEEP_TMP`              | Preserve disposable server state and logs          | `1`               |

When a black-box scenario fails, rerun the individual script with the same seed and `AMAQUET_KEEP_TMP=1` so `server.log`, configuration, AOF, and admin state remain available for diagnosis.

## Release recommendation

For pull requests, use bounded smoke settings. Before a release, use a longer run such as:

```bash
AMAQUET_FUZZ_SEED=4242 \
AMAQUET_SCENARIO_ITERATIONS=100 \
AMAQUET_FUZZ_ITERATIONS=250 \
AMAQUET_FRAME_FUZZ_ITERATIONS=5000 \
./tests/bash/run_developer_fuzz.sh

AMAQUET_GO_FUZZ_TIME=30s ./scripts/run_go_fuzz.sh
```

Run `go test -race ./...` separately because the race detector and external black-box workloads find different classes of defects.
