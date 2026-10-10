---
title: "Type construction and operation reference"
description: "Reference the Amaquet protocol contract for constructing and operating on typed values, including arguments, defaults, results, persistence, and failures."
---

This is the complete protocol contract for constructing and operating on Amaquet's 91 registered types. The individual type pages provide copyable creation examples; this page defines arguments, defaults, results, and the implementation behavior behind them.

## `OP` envelope

Use this command envelope to invoke a type-specific operation on an existing key.

```json
{
  "command": "OP",
  "args": {
    "key": "jobs",
    "operation": "ENQUEUE",
    "args": {
      "value": { "type": "json", "value": { "id": "job-1" } }
    }
  }
}
```

Operation names are case-insensitive at dispatch. Argument names are case-sensitive JSON object keys. A nested typed value always uses `{"type":"…","value":…}`; see [Wire values](../protocol/wire-values.md).

Read operations do not increment the key version. A successful mutation reserves approximate memory first and increments the version after the type object changes. A failed validation or operation does not increment it. Empty reads generally return `null`, an empty collection, or `false` as documented below.

Read-classified paths can still perform internal lazy maintenance. In particular, reliable-queue `STATS` and snapshots requeue or dead-letter expired leases without a version increment or a separate AOF mutation record.

The word `GET` in a directly encoded type's individual page means the top-level key command. It is not automatically an `OP GET`. Only operations listed in the tables below are accepted by `OP`.

## Construction

Direct values use `SET`; their JSON representations are defined in [Wire values](../protocol/wire-values.md). `json`, `messagepack`, and `cbor` support both direct `SET` and an empty `CREATE` initializer. Other composite/specialized values use:

```json
{
  "command": "CREATE",
  "args": {
    "key": "example",
    "type": "list",
    "options": {},
    "ttl_ms": 60000,
    "nx": true
  }
}
```

`CREATE` and `SET` both accept optional `ttl_ms`, `nx`, and `xx`. `nx` fails if a live key exists; `xx` fails if no live key exists. Supplying both is legal to decode but can never produce a useful conditional write. `expires_at_unix_nano` is the absolute-time form used by deterministic replay and advanced clients.

### Collection and structured options

These creation options define the initial shape and constraints of collection and structured values.

| Type                                                                                         | Options and defaults                                      | Constraints                                                      |
| -------------------------------------------------------------------------------------------- | --------------------------------------------------------- | ---------------------------------------------------------------- |
| `list`, `set`, `sorted_set`, `ordered_set`, `multiset`, `hashmap`, `ordered_map`, `multimap` | none                                                      | Start empty                                                      |
| `array`                                                                                      | `typed:false`, `element_type` omitted                     | A typed array requires each nested value to match `element_type` |
| `deque`                                                                                      | `capacity:16`                                             | Values below 8 become 8; grows by doubling                       |
| `ring_buffer`                                                                                | `capacity:1024`                                           | Minimum 1; overwrites the oldest value when full                 |
| `tuple`                                                                                      | `values:[]`                                               | Array of nested typed values; fixed after creation               |
| `json`                                                                                       | none                                                      | Starts as `{}`                                                   |
| `messagepack`, `cbor`                                                                        | none                                                      | Start as empty opaque bytes                                      |
| `record`                                                                                     | `schema:[]`                                               | Each field has `name`, registered `type`, and `nullable`         |
| `matrix`                                                                                     | `rows:1`, `cols:1`, `dtype:"float64"`, zero-filled `data` | `data` length must equal rows × columns                          |
| `tensor`                                                                                     | `shape:[1]`, `dtype:"float64"`, zero-filled `data`        | Every dimension positive; product must equal data length         |

The type package defines dtype constants `float64`, `float32`, `int64`, and `int32`. The constructor currently stores any supplied dtype string without validating it, and values are represented internally as `float64`; use the defined constants for portable data because dtype is descriptive metadata rather than an enforced conversion on every assignment.

### Queue, stream, and messaging options

Messaging structures begin empty unless their row identifies retention, delivery-attempt, or group-name state. These values are creation-time options; runtime behavior is controlled through the operations documented later on this page.

| Type                                                                                                             | Options and defaults | Constraints                                                                  |
| ---------------------------------------------------------------------------------------------------------------- | -------------------- | ---------------------------------------------------------------------------- |
| `fifo_queue`, `lifo_stack`, `priority_queue`, `blocking_queue`, `delayed_queue`, `stream`, `pubsub`, `event_log` | none                 | Start empty                                                                  |
| `reliable_queue`                                                                                                 | `max_attempts:5`     | Values below 1 become 5                                                      |
| `consumer_group`                                                                                                 | `name:"default"`     | Standalone snapshot object; stream groups are normally created on a `stream` |
| `persistent_topic`                                                                                               | `max_messages:0`     | `0` means no retention-count cap                                             |

### Bit and probabilistic options

Bit and probabilistic structures allocate or grow their internal state from these sizing parameters. Defaults favor general-purpose use, while constructor validation or fallback behavior is listed explicitly in the constraints column.

| Type                                    | Options and defaults                          | Constraints                                                  |
| --------------------------------------- | --------------------------------------------- | ------------------------------------------------------------ |
| `bitmap`, `bit_set`, `roaring_bitmap`   | none                                          | Grow as values/bits are added                                |
| `bit_field`                             | `size_bytes:0`                                | Negative values become 0; later writes grow it               |
| `hyperloglog`                           | `precision:14`                                | Precision 4…18                                               |
| `bloom_filter`, `counting_bloom_filter` | `expected:100000`, `false_positive_rate:0.01` | Expected > 0; rate strictly between 0 and 1                  |
| `cuckoo_filter`                         | `capacity:100000`                             | Allocates power-of-two four-slot buckets sized from capacity |
| `count_min_sketch`                      | `width:2048`, `depth:5`                       | Both positive                                                |
| `top_k`                                 | `k:10`                                        | Values below 1 become 1                                      |
| `t_digest`                              | `compression:100`                             | Values below 20 become 100                                   |

### Time-series, geo, vector, and index options

These constructors define retention, aggregation, dimensions, distance metrics, and index tuning. Some string options are stored at creation and validated only when an operation needs them, as noted after the table.

| Type                                                                    | Options and defaults                                               | Constraints                                        |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------ | -------------------------------------------------- |
| `timeseries`, `counter_series`, `gauge_series`                          | `labels:{}`, `retention_ms:0`                                      | `0` means no retention trimming                    |
| `histogram`                                                             | `bounds:[]`                                                        | Bounds are sorted; one overflow bucket is added    |
| `timeseries_labels`                                                     | `labels:{}`                                                        | Mutable string map                                 |
| `aggregated_series`                                                     | `bucket_ms:60000`, `aggregation:"avg"`                             | Source key is supplied at read time                |
| `geo_spatial_index`, `geospatial_secondary_index`                       | none                                                               | Start empty                                        |
| `vector_set`, `flat_vector_index`                                       | required positive `dim`, `metric:"cosine"`                         | Metrics: `cosine`, `dot`, `euclidean`, `manhattan` |
| `vector_metadata`                                                       | none                                                               | Starts as an empty JSON-like map                   |
| `hnsw_index`                                                            | required positive `dim`, `metric:"cosine"`, `m:16`, `ef_search:64` | `m<2` becomes 16; `ef_search<m` becomes 64         |
| `btree_index`                                                           | `degree:16`                                                        | Degree below 2 becomes 16                          |
| `hash_index`, `radix_tree`, `trie`, `inverted_index`, `secondary_index` | none                                                               | Start empty                                        |
| `full_text_document`                                                    | `id:""`, `metadata:null`                                           | Field arrays start empty                           |
| `tag_field`                                                             | `name:""`, `values:[]`                                             | Standalone field value                             |
| `numeric_field`                                                         | `name:""`, `value:0`                                               | Standalone field value                             |
| `text_field`                                                            | `name:""`, `value:""`, `weight:1`                                  | Standalone field value                             |

The portable aggregation names are `avg`, `sum`, `min`, `max`, `count`, `first`, and `last`; an `aggregated_series` stores the creation string and reports an unknown mode when a read tries to use it. Likewise, vector constructors store the metric string, while distance/search evaluation rejects anything other than `cosine`, `dot`, `euclidean`, or `manhattan`.

### Graph, synchronization, and network options

These options initialize graph, synchronization, and rate-limiting values before their first operation.

| Type                             | Options and defaults                   | Constraints                             |
| -------------------------------- | -------------------------------------- | --------------------------------------- |
| `adjacency_set`, `lease`, `lock` | none                                   | Start empty/unheld                      |
| `atomic_counter`                 | `initial:0`                            | Signed 64-bit                           |
| `barrier`                        | `target:1`                             | Positive participant count              |
| `semaphore`                      | `limit:1`                              | Positive permit count                   |
| `rate_limit_bucket`              | `capacity:100`, `refill_per_second:10` | Negative values clamp to 0; starts full |

`geo_point`, `bounding_box`, `polygon`, `dense_vector`, `sparse_vector`, `quantized_vector`, `cidr_network`, and `url` are direct `SET` values rather than `CREATE` types.

## Scalar and direct-value operations

The following operations mutate or inspect scalar values that are normally created with `SET`.

| Type                      | Operation        | Arguments                                          | Result and behavior                                                  |
| ------------------------- | ---------------- | -------------------------------------------------- | -------------------------------------------------------------------- |
| `integer`                 | `ADD`            | `delta` signed integer                             | New signed value                                                     |
| `unsigned_integer`        | `ADD`            | `delta` unsigned integer                           | New unsigned value                                                   |
| `float`                   | `ADD`            | `delta` number                                     | New float64 value                                                    |
| `decimal`, `big_decimal`  | `ADD`, `SUB`     | `value` decimal string                             | Exact result string; scales are aligned before arithmetic            |
| `decimal`, `big_decimal`  | `CMP`            | `value` decimal string                             | `-1`, `0`, or `1`; read-only                                         |
| `boolean`                 | `TOGGLE`         | none                                               | New boolean                                                          |
| `duration`                | `ADD`            | `delta_ms` integer milliseconds or duration string | New Go-duration string                                               |
| `big_integer`             | `ADD`            | `value` base-10 integer string                     | New arbitrary-precision integer string                               |
| `geo_point`               | `DISTANCE`       | `point:{lat,lon}`                                  | Haversine distance in meters using radius 6,371,008.8 m              |
| `bounding_box`, `polygon` | `CONTAINS`       | `point:{lat,lon}`                                  | Boolean containment; polygon uses ray casting on lon/lat coordinates |
| `dense_vector`            | `DISTANCE`       | `vector:[...]`, optional `metric:"cosine"`         | Numeric distance; dimensions must match                              |
| `quantized_vector`        | `DEQUANTIZE`     | none                                               | Float32 vector `(q-zero_point)*scale`                                |
| `cidr_network`            | `CONTAINS`       | `ip` string                                        | Boolean parsed-IP containment                                        |
| `url`                     | `HOST`, `SCHEME` | none                                               | Parsed host or scheme string                                         |

`binary_string`, `utf8_string`, `null`, `timestamp`, `uuid`, `symbol`, and `sparse_vector` have no type-specific `OP`; use the normal key commands. Standalone sparse-vector distance is not exposed by the current dispatcher.

Signed and unsigned fixed-width additions (`integer`, `unsigned_integer`, `atomic_counter`, and the millisecond addition used by `duration`) use Go's native fixed-width arithmetic and wrap on overflow. `big_integer` and the decimal types use arbitrary-precision representations instead.

Vector metric results are distances sorted ascending: cosine is `1-cosine_similarity`, dot is the negative dot product, Euclidean is L2, and Manhattan is L1. A zero vector has cosine distance 1.

## Collection operations

These operations manage ordered, unordered, map-like, and fixed-shape collection values.

| Type          | Operation                   | Arguments                             | Result and behavior                                               |
| ------------- | --------------------------- | ------------------------------------- | ----------------------------------------------------------------- |
| `list`        | `PUSH_FRONT`, `PUSH_BACK`   | nested `value`                        | New length                                                        |
| `list`        | `POP_FRONT`, `POP_BACK`     | none                                  | Typed value or `null`                                             |
| `list`        | `RANGE`                     | `start:0`, `stop:-1`                  | Inclusive slice; negative indexes count from end                  |
| `list`        | `LEN`                       | none                                  | Length                                                            |
| `array`       | `APPEND`                    | nested `value`                        | New length; enforces typed-array element type                     |
| `array`       | `SET`                       | `index`, nested `value`               | `true`; negative indexes allowed; out of range errors             |
| `array`       | `GET`                       | `index`                               | Typed value or `null`; negative indexes allowed                   |
| `array`       | `VALUES`, `LEN`             | none                                  | Copy of values or array length                                    |
| `deque`       | `PUSH_FRONT`, `PUSH_BACK`   | nested `value`                        | New length                                                        |
| `deque`       | `POP_FRONT`, `POP_BACK`     | none                                  | Typed value or `null`                                             |
| `deque`       | `VALUES`, `LEN`             | none                                  | Logical-order values or length                                    |
| `ring_buffer` | `PUSH`                      | nested `value`                        | Current length; overwrites oldest at capacity                     |
| `ring_buffer` | `VALUES`, `LEN`             | none                                  | Oldest-to-newest values or length                                 |
| `tuple`       | `GET`                       | `index`                               | Typed value or `null`; negative indexes allowed                   |
| `tuple`       | `VALUES`, `LEN`             | none                                  | Fixed values or tuple length                                      |
| `set`         | `ADD`, `REMOVE`, `CONTAINS` | nested `value`                        | Boolean added/removed/present                                     |
| `set`         | `MEMBERS`, `LEN`            | none                                  | Members or count; member order is unspecified                     |
| `sorted_set`  | `ADD`                       | nested `member`, numeric `score`      | `true` if new, `false` if score replaced                          |
| `sorted_set`  | `REMOVE`, `SCORE`           | nested `member`                       | Removed boolean; score or `null`                                  |
| `sorted_set`  | `RANGE`                     | `start:0`, `stop:-1`, `reverse:false` | Inclusive score-ordered entries; value identity breaks score ties |
| `ordered_set` | `ADD`, `REMOVE`             | nested `value`                        | Boolean; re-adding an existing value does not move it             |
| `ordered_set` | `VALUES`                    | none                                  | Insertion-order values                                            |
| `multiset`    | `ADD`, `REMOVE`             | nested `value`, optional `count:1`    | Remaining/new unsigned count                                      |
| `multiset`    | `COUNT`                     | nested `value`                        | Current count                                                     |
| `hashmap`     | `SET`                       | `field`, nested `value`               | `true`                                                            |
| `hashmap`     | `GET`                       | `field`                               | Typed value or `null`                                             |
| `hashmap`     | `DELETE`                    | `field`                               | Removed boolean                                                   |
| `hashmap`     | `KEYS`                      | none                                  | Lexically sorted field names                                      |
| `ordered_map` | `SET`                       | `field`, nested `value`               | `true`; replacement retains original position                     |
| `ordered_map` | `GET`                       | `field`                               | Typed value or `null`                                             |
| `ordered_map` | `DELETE`                    | `field`                               | Removed boolean                                                   |
| `ordered_map` | `ENTRIES`                   | none                                  | Insertion-order key/value entries                                 |
| `multimap`    | `ADD`                       | `field`, nested `value`               | `true`; duplicates are allowed                                    |
| `multimap`    | `GET`                       | `field`                               | Copy of all values                                                |
| `multimap`    | `REMOVE`                    | `field`, nested `value`               | Removes the first equal value; boolean                            |

Lists use Go slices, so front insertion/removal is linear. Deques use a growable circular buffer. Sets use a deterministic typed-value key. Sorted-set reads sort current map entries by score rather than maintaining a skip list.

## Structured operations

These operations read and modify records, documents, matrices, tensors, and other structured values.

| Type               | Operation | Arguments                        | Result and behavior                      |
| ------------------ | --------- | -------------------------------- | ---------------------------------------- |
| `json`             | `GET`     | none                             | Deep-copied JSON document                |
| `json`             | `SET`     | arbitrary JSON `value`           | `true`; replaces the whole document      |
| `record`           | `SET`     | `field`, nested `value`          | `true`; schema/type/nullability enforced |
| `record`           | `GET`     | `field`                          | Typed value or `null`                    |
| `matrix`, `tensor` | `GET`     | `indices:[...]`                  | Numeric cell; rank and bounds enforced   |
| `matrix`, `tensor` | `SET`     | `indices:[...]`, numeric `value` | `true`                                   |
| `matrix`, `tensor` | `LEN`     | none                             | Flat element count                       |

`messagepack` and `cbor` expose only top-level `GET`; they are opaque byte containers and are not decoded by Amaquet.

## Queue operations

These operations enqueue, dequeue, inspect, and coordinate queue-like values.

| Type             | Operation         | Arguments                                                 | Result and behavior                                                         |
| ---------------- | ----------------- | --------------------------------------------------------- | --------------------------------------------------------------------------- |
| `fifo_queue`     | `ENQUEUE`         | nested `value`                                            | New length                                                                  |
| `fifo_queue`     | `DEQUEUE`, `PEEK` | none                                                      | Oldest typed value or `null`                                                |
| `fifo_queue`     | `LEN`             | none                                                      | Length                                                                      |
| `lifo_stack`     | `PUSH`            | nested `value`                                            | New length                                                                  |
| `lifo_stack`     | `POP`, `PEEK`     | none                                                      | Newest typed value or `null`                                                |
| `lifo_stack`     | `LEN`             | none                                                      | Length                                                                      |
| `priority_queue` | `ENQUEUE`         | nested `value`, numeric `priority`                        | New length                                                                  |
| `priority_queue` | `DEQUEUE`         | none                                                      | `{value,priority,sequence}` or `null`; higher priority first, FIFO for ties |
| `priority_queue` | `LEN`             | none                                                      | Length                                                                      |
| `blocking_queue` | `ENQUEUE`         | nested `value`                                            | `true`; errors after close                                                  |
| `blocking_queue` | `DEQUEUE`         | none                                                      | Waits for a value, close, cancellation, or server command deadline          |
| `blocking_queue` | `CLOSE`           | none                                                      | `true`; wakes a waiter; close is permanent                                  |
| `blocking_queue` | `LEN`             | none                                                      | Length                                                                      |
| `delayed_queue`  | `ENQUEUE`         | `id`, nested `value`, `available_at_ms` Unix milliseconds | New length                                                                  |
| `delayed_queue`  | `POP_READY`       | optional replay-only `now_unix_nano`                      | Earliest ready item, FIFO sequence for equal times, or `null`               |
| `delayed_queue`  | `LEN`             | none                                                      | Length                                                                      |
| `reliable_queue` | `ENQUEUE`         | `id`, nested `value`                                      | `true`; IDs are not checked for uniqueness                                  |
| `reliable_queue` | `CLAIM`           | optional `lease_ms:30000`                                 | Oldest ready message or `null`; increments attempts                         |
| `reliable_queue` | `ACK`, `NACK`     | `id`                                                      | Boolean; NACK requeues or dead-letters at `max_attempts`                    |
| `reliable_queue` | `STATS`           | none                                                      | `{ready,pending,dead}` after expired leases are requeued/dead-lettered      |

Delayed-queue ready selection scans the current items. Reliable-queue expired leases are processed lazily during claim, stats, and snapshot operations.

## Stream and messaging operations

These operations append, consume, acknowledge, and inspect stream and messaging values.

| Type               | Operation      | Arguments                                                      | Result and behavior                                                          |
| ------------------ | -------------- | -------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `stream`           | `ADD`          | `fields` map of nested values; optional `id:"millis-sequence"` | Monotonic stream ID; server generates a deterministic ID when omitted        |
| `stream`           | `RANGE`        | optional `count:100`                                           | First entries up to count; current protocol does not expose start/end bounds |
| `stream`           | `GROUP_CREATE` | `group`                                                        | `true`; starts before the first entry                                        |
| `stream`           | `GROUP_READ`   | `group`, `consumer`, optional `count:10`                       | New entries after group's last-delivered ID and records them pending         |
| `stream`           | `ACK`          | `group`, `ids:["…"]`                                           | Number removed from pending                                                  |
| `stream`           | `PENDING`      | `group`                                                        | Pending records sorted by stream ID                                          |
| `stream`           | `LEN`          | none                                                           | Entry count                                                                  |
| `consumer_group`   | `SNAPSHOT`     | none                                                           | Standalone group's name, cursor, and pending entries                         |
| `pubsub`           | `PUBLISH`      | `channel`, nested `value`                                      | Number of server subscriber buffers that accepted the message                |
| `pubsub`           | `STATS`        | none                                                           | Cumulative `{published,delivered,dropped}`                                   |
| `persistent_topic` | `PUBLISH`      | optional `message_key`, nested `value`                         | Monotonic zero-based offset                                                  |
| `persistent_topic` | `READ`         | required `from`, optional `count:100`                          | Messages with offset ≥ `from`                                                |
| `event_log`        | `APPEND`       | `event_type`, optional `aggregate_id`, nested `payload`        | Event with one-based sequence and UTC timestamp                              |
| `event_log`        | `READ`         | optional `from:1`, `count:100`                                 | Events with sequence ≥ `from`                                                |
| `event_log`        | `LEN`          | none                                                           | Event count                                                                  |

Pub/Sub subscriptions do not use `OP`. Send the dedicated `SUBSCRIBE` opcode with `key`, `channel`, and optional `buffer` (default 16 on the server; the Go helper defaults to 64). Delivery is best-effort: a full subscriber buffer drops that event instead of blocking publishers. `persistent_topic` and `event_log` retain messages as database values; Pub/Sub does not.

## Bit-structure operations

These operations manipulate individual bits, ranges, and compact bit-oriented indexes.

| Type                | Operation                   | Arguments                             | Result and behavior                                            |
| ------------------- | --------------------------- | ------------------------------------- | -------------------------------------------------------------- |
| `bitmap`, `bit_set` | `SET`                       | unsigned `bit`, optional `value:true` | Previous bit value; storage grows to the addressed 64-bit word |
| `bitmap`, `bit_set` | `GETBIT`                    | unsigned `bit`                        | Boolean; out-of-range reads are false                          |
| `bitmap`, `bit_set` | `COUNT`                     | none                                  | Population count                                               |
| `bit_field`         | `SET`                       | `offset`, `width`, `value` unsigned   | `true`; width must be 1…64; bits are big-endian within bytes   |
| `bit_field`         | `GET`                       | `offset`, `width`                     | Unsigned value; an unread allocated range returns 0            |
| `roaring_bitmap`    | `ADD`, `REMOVE`, `CONTAINS` | unsigned `value` ≤ 2³²−1              | Boolean                                                        |
| `roaring_bitmap`    | `VALUES`                    | none                                  | Sorted uint32 values                                           |
| `roaring_bitmap`    | `CARDINALITY`               | none                                  | Count                                                          |

Roaring values are partitioned by the high 16 bits. Each low-16-bit container adapts among sorted arrays, 65,536-bit bitmaps, and run containers. Dense arrays switch around 4,096 values; long consecutive ranges prefer run storage.

## Probabilistic operations

These operations update and query approximate-membership, cardinality, frequency, and percentile structures.

| Type                    | Operation  | Arguments                                   | Result and behavior                                                                                               |
| ----------------------- | ---------- | ------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `hyperloglog`           | `ADD`      | string `value`                              | Whether a register increased                                                                                      |
| `hyperloglog`           | `COUNT`    | none                                        | Rounded approximate cardinality with small-range linear counting                                                  |
| `bloom_filter`          | `ADD`      | string `value`                              | Whether any filter bit changed                                                                                    |
| `bloom_filter`          | `CONTAINS` | string `value`                              | Probabilistic membership                                                                                          |
| `counting_bloom_filter` | `ADD`      | string `value`                              | `true`; saturating uint16 counters                                                                                |
| `counting_bloom_filter` | `REMOVE`   | string `value`                              | Boolean; decrements only if every counter is nonzero                                                              |
| `counting_bloom_filter` | `CONTAINS` | string `value`                              | Probabilistic membership                                                                                          |
| `cuckoo_filter`         | `ADD`      | string `value`                              | Boolean; insertion can fail after 500 relocations                                                                 |
| `cuckoo_filter`         | `REMOVE`   | string `value`                              | Boolean indicating whether a matching fingerprint was removed                                                     |
| `cuckoo_filter`         | `CONTAINS` | string `value`                              | Probabilistic membership                                                                                          |
| `count_min_sketch`      | `ADD`      | string `value`, optional unsigned `count:1` | Current minimum counter estimate                                                                                  |
| `count_min_sketch`      | `ESTIMATE` | string `value`                              | Approximate frequency (never below inserted count absent overflow)                                                |
| `top_k`                 | `ADD`      | string `value`, optional unsigned `count:1` | Count-Min estimate                                                                                                |
| `top_k`                 | `ITEMS`    | none                                        | Up to `k` candidates by estimated count, then lexical tie-break                                                   |
| `t_digest`              | `ADD`      | numeric `value`, optional `weight:1`        | `true`; non-positive weights are ignored internally                                                               |
| `t_digest`              | `QUANTILE` | numeric `q`                                 | Approximate quantile; q clamps to 0…1; empty digest returns JSON-incompatible NaN and therefore should be avoided |

All hashing in these probabilistic structures derives 64-bit values from SHA-256 with explicit seeds. Bloom filters use double hashing. The Cuckoo filter uses four 16-bit fingerprints per bucket. Top-K retains at most roughly `4*k` candidates over a fixed 2048×5 Count-Min Sketch. These structures provide estimates/false-positive tradeoffs, not exact sets or counters.

## Time-series operations

These operations append observations and query retained time-series and aggregated data.

| Type                | Operation            | Arguments                                                       | Result and behavior                                                |
| ------------------- | -------------------- | --------------------------------------------------------------- | ------------------------------------------------------------------ |
| `timeseries`        | `ADD`                | optional `timestamp` Unix ms (default now), numeric `value`     | `true`; replaces an existing point at the same timestamp           |
| `counter_series`    | `ADD`                | optional `timestamp`, numeric `value` delta                     | `true`; stores last value plus delta                               |
| `gauge_series`      | `SET`                | optional `timestamp`, numeric `value`                           | `true`; stores the supplied value                                  |
| all three series    | `RANGE`              | optional `from:0`, `to:now` Unix ms                             | Inclusive sorted points                                            |
| all three series    | `LAST`               | none                                                            | Last point or `null`                                               |
| all three series    | `AGGREGATE`          | optional `from`, `to`, `bucket_ms:60000`, `aggregation:"avg"`   | Bucket-start/value points                                          |
| all three series    | `LEN`                | none                                                            | Point count                                                        |
| `histogram`         | `OBSERVE`            | numeric `value`                                                 | `true`; increments first bound ≥ value, otherwise overflow bucket  |
| `histogram`         | `SNAPSHOT`           | none                                                            | `{bounds,counts,count,sum}`                                        |
| `timeseries_labels` | `GET`                | none                                                            | Copy of label map                                                  |
| `timeseries_labels` | `SET`                | `label`, string `value`                                         | `true`                                                             |
| `aggregated_series` | `RANGE`, `AGGREGATE` | `source_key`; optional `from`, `to`, `bucket_ms`, `aggregation` | Aggregates a `timeseries`, `counter_series`, or `gauge_series` key |

Aggregation modes are `avg`, `sum`, `min`, `max`, `count`, `first`, and `last`. Buckets are anchored at `from`. Retention trimming occurs on writes and uses the server's current time, not the newest sample time; old out-of-order points can therefore be removed immediately.

## Geospatial operations

These operations store geographic members and query positions, bounds, and radius matches.

| Type                                              | Operation | Arguments                                               | Result and behavior                         |
| ------------------------------------------------- | --------- | ------------------------------------------------------- | ------------------------------------------- |
| `geo_spatial_index`, `geospatial_secondary_index` | `ADD`     | `id`, `point:{lat,lon}`, optional string `metadata` map | `true`; replacing an ID moves its grid cell |
| both indexes                                      | `REMOVE`  | `id`                                                    | Boolean                                     |
| both indexes                                      | `RADIUS`  | `center`, `radius_meters`, optional `limit:100`         | Distance-sorted hits with `distance_meters` |
| both indexes                                      | `BOX`     | `box:{minLat,minLon,maxLat,maxLon}`                     | ID-sorted items inside inclusive bounds     |
| both indexes                                      | `POLYGON` | `points:[...]`                                          | ID-sorted items passing point-in-polygon    |

The index uses 0.25-degree latitude/longitude grid cells to find candidates, then applies exact Haversine/box/ray-casting checks. Queries whose bounding box spans more than 100,000 cells fall back to scanning all items. Polygons are planar lon/lat ray casts and do not add special antimeridian wrapping.

## Vector operations

These operations manage vectors and run exact or approximate similarity searches.

| Type                              | Operation | Arguments                           | Result and behavior                                  |
| --------------------------------- | --------- | ----------------------------------- | ---------------------------------------------------- |
| `vector_set`, `flat_vector_index` | `ADD`     | `id`, `vector`, optional `metadata` | `true`; vector dimension enforced; replaces same ID  |
| both                              | `REMOVE`  | `id`                                | Boolean                                              |
| both                              | `SEARCH`  | `vector`, optional `k:10`           | Exact full-scan results sorted by ascending distance |
| both                              | `LEN`     | none                                | Item count                                           |
| `hnsw_index`                      | `ADD`     | `id`, `vector`, optional `metadata` | `true`; replaces same ID and reconnects graph        |
| `hnsw_index`                      | `REMOVE`  | `id`                                | Boolean                                              |
| `hnsw_index`                      | `SEARCH`  | `vector`, optional `k:10`           | Approximate ascending-distance neighbors             |
| `vector_metadata`                 | `GET`     | none                                | Copy of metadata map                                 |
| `vector_metadata`                 | `SET`     | `field`, arbitrary JSON `value`     | `true`                                               |
| `vector_metadata`                 | `DELETE`  | `field`                             | Whether a field existed                              |

The current protocol does not expose metadata-filter expressions even though the internal search API accepts a filter callback. HNSW assigns deterministic levels from FNV-1a of the item ID, uses bidirectional edges, caps degree at `m` above level 0 and `2*m` at level 0, and searches with configured `ef_search` (raised to at least `k`). Exact vector sets scan all items.

## Index and search operations

These operations populate and query indexes, trees, and full-text search structures.

| Type                 | Operation         | Arguments                                                           | Result and behavior                                  |
| -------------------- | ----------------- | ------------------------------------------------------------------- | ---------------------------------------------------- |
| `btree_index`        | `INSERT`          | string `value`, `id`                                                | `true`                                               |
| `btree_index`        | `DELETE`          | string `value`, `id`                                                | Removed boolean                                      |
| `btree_index`        | `SEARCH`          | string `value`                                                      | Lexically sorted IDs for exact key                   |
| `btree_index`        | `RANGE`           | string `value` as inclusive minimum; optional `max:""`, `limit:100` | Map of key to sorted IDs; limit counts keys          |
| `hash_index`         | `ADD`             | string `value`, `id`                                                | `true`                                               |
| `hash_index`         | `REMOVE`          | string `value`, `id`                                                | Removed boolean                                      |
| `hash_index`         | `LOOKUP`          | string `value`                                                      | Sorted IDs                                           |
| `radix_tree`, `trie` | `INSERT`          | string `value`, `id`                                                | `true`                                               |
| both                 | `EXACT`           | string `value`                                                      | Sorted IDs                                           |
| both                 | `PREFIX`          | string `prefix`, optional `limit:100`                               | Lexically traversed map of matching strings to IDs   |
| `inverted_index`     | `INDEX`           | `id`, string `text`                                                 | `true`; replaces previous terms for ID               |
| `inverted_index`     | `REMOVE`          | `id`                                                                | `true` even if absent                                |
| `inverted_index`     | `SEARCH`          | string `query`, optional `limit:20`                                 | Hits sorted by descending TF×IDF-like score, then ID |
| `secondary_index`    | `ADD`             | `field`, string `value`, `id`                                       | `true`                                               |
| `secondary_index`    | `LOOKUP`          | `field`, string `value`                                             | Sorted IDs                                           |
| `full_text_document` | `ADD_TEXT`        | `name`, string `value`, optional `weight:1`                         | `true`; appends a field                              |
| `full_text_document` | `ADD_TAG`         | `name`, `values` string array                                       | `true`; appends a field                              |
| `full_text_document` | `ADD_NUMERIC`     | `name`, numeric `value`                                             | `true`; appends a field                              |
| `full_text_document` | `GET`, `SNAPSHOT` | none                                                                | Complete document snapshot                           |
| `tag_field`          | `SET`             | optional `name`, optional `values`                                  | `true`; replaces supplied properties                 |
| `tag_field`          | `GET`             | none                                                                | Field object                                         |
| `numeric_field`      | `SET`             | optional `name`, optional numeric `value`                           | `true`                                               |
| `numeric_field`      | `GET`             | none                                                                | Field object                                         |
| `text_field`         | `SET`             | optional `name`, `value`, `weight`                                  | `true`                                               |
| `text_field`         | `GET`             | none                                                                | Field object                                         |

The B-tree uses minimum degree `t`, at most `2t-1` keys per node, and rebuilds the tree after deletion. Radix-tree/trie traversal is byte-oriented UTF-8, not Unicode-code-point compressed. The inverted index lowercases and tokenizes Unicode letter/digit runs, records term frequency, and ranks query terms with an IDF-like natural-log approximation. `full_text_document` is a field container; it does not automatically update an `inverted_index`.

## Graph, synchronization, and network operations

These operations manage relationships and coordinate concurrent clients through synchronization and rate-limit values.

| Type                | Operation          | Arguments                                 | Result and behavior                                                                                                                     |
| ------------------- | ------------------ | ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `adjacency_set`     | `ADD`              | `from`, `to`, optional numeric `weight:1` | `true`; directed edge                                                                                                                   |
| `adjacency_set`     | `REMOVE`           | `from`, `to`                              | Removed boolean; directed edge                                                                                                          |
| `adjacency_set`     | `NEIGHBORS`        | `from`                                    | Map of outgoing neighbor to weight                                                                                                      |
| `atomic_counter`    | `ADD`              | signed `delta`                            | New value                                                                                                                               |
| `atomic_counter`    | `LOAD`             | none                                      | Current value                                                                                                                           |
| `atomic_counter`    | `CAS`              | signed `old`, signed `new`                | Swap-success boolean                                                                                                                    |
| `lease`             | `ACQUIRE`          | `owner`, `token`, optional `ttl_ms:30000` | Boolean; only when absent/expired and TTL positive                                                                                      |
| `lease`             | `RENEW`, `RELEASE` | `token`; renew accepts `ttl_ms:30000`     | Boolean token-checked result                                                                                                            |
| `lease`             | `HELD`             | none                                      | Whether a non-expired token exists                                                                                                      |
| `lock`              | `TRY_LOCK`         | `owner`, `token`, optional `ttl_ms:30000` | Boolean; implemented by a lease                                                                                                         |
| `lock`              | `RENEW`, `UNLOCK`  | `token`; renew accepts `ttl_ms:30000`     | Boolean                                                                                                                                 |
| `barrier`           | `WAIT`             | none                                      | `true` when the target arrives; reusable across generations, and cancellation removes an arrival only from its still-current generation |
| `semaphore`         | `ACQUIRE`          | none                                      | `true` after a permit is obtained; context-aware wait                                                                                   |
| `semaphore`         | `TRY_ACQUIRE`      | none                                      | Immediate boolean                                                                                                                       |
| `semaphore`         | `RELEASE`          | none                                      | Boolean; false when no permit is held                                                                                                   |
| `semaphore`         | `STATS`            | none                                      | `{limit,in_use}`                                                                                                                        |
| `rate_limit_bucket` | `ALLOW`            | optional numeric `cost:1`                 | Boolean; non-positive cost is allowed without consuming tokens                                                                          |
| `rate_limit_bucket` | `REMAINING`        | none                                      | Current tokens after time-based refill                                                                                                  |

Lease/lock ownership is enforced by caller-supplied tokens, not by the authenticated actor automatically. Barrier and semaphore waits are bounded by request cancellation and `limits.command_timeout_ms`. Lease, lock, barrier, semaphore, Pub/Sub, and rate-limit-bucket mutations are deliberately excluded from AOF journaling because restoring live coordination state can recreate stale ownership.

`cidr_network` and `url` are direct values; their operations are in the scalar table above.

## Persistence classification

All `SET`, `CREATE`, `CAS`, `DEL`, `EXPIRE`, `PERSIST`, and blob-upload mutations are journaled when AOF is enabled. Mutating `OP` calls are journaled except for the process-local coordination types just listed. The dispatcher resolves generated IDs, timestamps, and relative TTLs before writing the prepare record so replay uses the historical value rather than the restart clock.

See [Persistence and recovery](../concepts/persistence.md) for the transaction and recovery format.
