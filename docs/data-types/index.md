---
title: "Data types"
description: "Browse Amaquet's 91 stable wire-level data types, their categories, construction commands, supported operations, and links to detailed references."
---

Amaquet exposes **91 stable wire-level types**. Type IDs are stable protocol/storage identities and are separate from internal encodings.

Use the [construction and operation reference](operations.md) for the complete argument, default, result, persistence, and implementation contract behind every operation listed here. Direct JSON encodings are defined in [Wire values and snapshots](../protocol/wire-values.md).

| Type                                                                | Category                            | Construction | Operations |
| ------------------------------------------------------------------- | ----------------------------------- | ------------ | ---------: |
| [`binary_string`](types/binary_string.md)                           | Scalar                              | SET          |          1 |
| [`utf8_string`](types/utf8_string.md)                               | Scalar                              | SET          |          1 |
| [`integer`](types/integer.md)                                       | Scalar                              | SET          |          1 |
| [`unsigned_integer`](types/unsigned_integer.md)                     | Scalar                              | SET          |          1 |
| [`float`](types/float.md)                                           | Scalar                              | SET          |          1 |
| [`decimal`](types/decimal.md)                                       | Scalar                              | SET          |          3 |
| [`boolean`](types/boolean.md)                                       | Scalar                              | SET          |          1 |
| [`null`](types/null.md)                                             | Scalar                              | SET          |          1 |
| [`timestamp`](types/timestamp.md)                                   | Scalar                              | SET          |          1 |
| [`duration`](types/duration.md)                                     | Scalar                              | SET          |          1 |
| [`uuid`](types/uuid.md)                                             | Scalar                              | SET          |          1 |
| [`big_integer`](types/big_integer.md)                               | Scalar                              | SET          |          1 |
| [`big_decimal`](types/big_decimal.md)                               | Scalar                              | SET          |          3 |
| [`symbol`](types/symbol.md)                                         | Scalar                              | SET          |          1 |
| [`list`](types/list.md)                                             | Collections                         | CREATE       |          6 |
| [`array`](types/array.md)                                           | Collections                         | CREATE       |          5 |
| [`deque`](types/deque.md)                                           | Collections                         | CREATE       |          6 |
| [`ring_buffer`](types/ring_buffer.md)                               | Collections                         | CREATE       |          3 |
| [`tuple`](types/tuple.md)                                           | Collections                         | CREATE       |          3 |
| [`set`](types/set.md)                                               | Collections                         | CREATE       |          5 |
| [`sorted_set`](types/sorted_set.md)                                 | Collections                         | CREATE       |          4 |
| [`ordered_set`](types/ordered_set.md)                               | Collections                         | CREATE       |          3 |
| [`multiset`](types/multiset.md)                                     | Collections                         | CREATE       |          3 |
| [`hashmap`](types/hashmap.md)                                       | Collections                         | CREATE       |          4 |
| [`ordered_map`](types/ordered_map.md)                               | Collections                         | CREATE       |          4 |
| [`multimap`](types/multimap.md)                                     | Collections                         | CREATE       |          3 |
| [`json`](types/json.md)                                             | Structured                          | CREATE / SET |          2 |
| [`messagepack`](types/messagepack.md)                               | Structured                          | CREATE / SET |          1 |
| [`cbor`](types/cbor.md)                                             | Structured                          | CREATE / SET |          1 |
| [`record`](types/record.md)                                         | Structured                          | CREATE       |          2 |
| [`matrix`](types/matrix.md)                                         | Structured                          | CREATE       |          3 |
| [`tensor`](types/tensor.md)                                         | Structured                          | CREATE       |          3 |
| [`fifo_queue`](types/fifo_queue.md)                                 | Queues and messaging                | CREATE       |          4 |
| [`lifo_stack`](types/lifo_stack.md)                                 | Queues and messaging                | CREATE       |          4 |
| [`priority_queue`](types/priority_queue.md)                         | Queues and messaging                | CREATE       |          3 |
| [`blocking_queue`](types/blocking_queue.md)                         | Queues and messaging                | CREATE       |          4 |
| [`delayed_queue`](types/delayed_queue.md)                           | Queues and messaging                | CREATE       |          3 |
| [`reliable_queue`](types/reliable_queue.md)                         | Queues and messaging                | CREATE       |          5 |
| [`stream`](types/stream.md)                                         | Queues and messaging                | CREATE       |          7 |
| [`consumer_group`](types/consumer_group.md)                         | Queues and messaging                | CREATE       |          1 |
| [`pubsub`](types/pubsub.md)                                         | Queues and messaging                | CREATE       |          2 |
| [`persistent_topic`](types/persistent_topic.md)                     | Queues and messaging                | CREATE       |          2 |
| [`event_log`](types/event_log.md)                                   | Queues and messaging                | CREATE       |          3 |
| [`bitmap`](types/bitmap.md)                                         | Bit structures                      | CREATE       |          3 |
| [`bit_field`](types/bit_field.md)                                   | Bit structures                      | CREATE       |          2 |
| [`bit_set`](types/bit_set.md)                                       | Bit structures                      | CREATE       |          3 |
| [`roaring_bitmap`](types/roaring_bitmap.md)                         | Bit structures                      | CREATE       |          5 |
| [`hyperloglog`](types/hyperloglog.md)                               | Probabilistic                       | CREATE       |          2 |
| [`bloom_filter`](types/bloom_filter.md)                             | Probabilistic                       | CREATE       |          2 |
| [`counting_bloom_filter`](types/counting_bloom_filter.md)           | Probabilistic                       | CREATE       |          3 |
| [`cuckoo_filter`](types/cuckoo_filter.md)                           | Probabilistic                       | CREATE       |          3 |
| [`count_min_sketch`](types/count_min_sketch.md)                     | Probabilistic                       | CREATE       |          2 |
| [`top_k`](types/top_k.md)                                           | Probabilistic                       | CREATE       |          2 |
| [`t_digest`](types/t_digest.md)                                     | Probabilistic                       | CREATE       |          2 |
| [`timeseries`](types/timeseries.md)                                 | Time series                         | CREATE       |          5 |
| [`counter_series`](types/counter_series.md)                         | Time series                         | CREATE       |          5 |
| [`gauge_series`](types/gauge_series.md)                             | Time series                         | CREATE       |          5 |
| [`histogram`](types/histogram.md)                                   | Time series                         | CREATE       |          2 |
| [`timeseries_labels`](types/timeseries_labels.md)                   | Time series                         | CREATE       |          2 |
| [`aggregated_series`](types/aggregated_series.md)                   | Time series                         | CREATE       |          2 |
| [`geo_point`](types/geo_point.md)                                   | Geospatial                          | SET          |          1 |
| [`geo_spatial_index`](types/geo_spatial_index.md)                   | Geospatial                          | CREATE       |          5 |
| [`bounding_box`](types/bounding_box.md)                             | Geospatial                          | SET          |          1 |
| [`polygon`](types/polygon.md)                                       | Geospatial                          | SET          |          1 |
| [`dense_vector`](types/dense_vector.md)                             | Vectors                             | SET          |          1 |
| [`sparse_vector`](types/sparse_vector.md)                           | Vectors                             | SET          |          1 |
| [`vector_set`](types/vector_set.md)                                 | Vectors                             | CREATE       |          4 |
| [`vector_metadata`](types/vector_metadata.md)                       | Vectors                             | CREATE       |          3 |
| [`quantized_vector`](types/quantized_vector.md)                     | Vectors                             | SET          |          1 |
| [`btree_index`](types/btree_index.md)                               | Indexes and search                  | CREATE       |          4 |
| [`hash_index`](types/hash_index.md)                                 | Indexes and search                  | CREATE       |          3 |
| [`radix_tree`](types/radix_tree.md)                                 | Indexes and search                  | CREATE       |          3 |
| [`trie`](types/trie.md)                                             | Indexes and search                  | CREATE       |          3 |
| [`inverted_index`](types/inverted_index.md)                         | Indexes and search                  | CREATE       |          3 |
| [`hnsw_index`](types/hnsw_index.md)                                 | Indexes and search                  | CREATE       |          3 |
| [`flat_vector_index`](types/flat_vector_index.md)                   | Indexes and search                  | CREATE       |          4 |
| [`geospatial_secondary_index`](types/geospatial_secondary_index.md) | Indexes and search                  | CREATE       |          5 |
| [`secondary_index`](types/secondary_index.md)                       | Indexes and search                  | CREATE       |          2 |
| [`full_text_document`](types/full_text_document.md)                 | Indexes and search                  | CREATE       |          5 |
| [`tag_field`](types/tag_field.md)                                   | Indexes and search                  | CREATE       |          2 |
| [`numeric_field`](types/numeric_field.md)                           | Indexes and search                  | CREATE       |          2 |
| [`text_field`](types/text_field.md)                                 | Indexes and search                  | CREATE       |          2 |
| [`adjacency_set`](types/adjacency_set.md)                           | Graph, synchronization, and network | CREATE       |          3 |
| [`atomic_counter`](types/atomic_counter.md)                         | Graph, synchronization, and network | CREATE       |          3 |
| [`lease`](types/lease.md)                                           | Graph, synchronization, and network | CREATE       |          4 |
| [`lock`](types/lock.md)                                             | Graph, synchronization, and network | CREATE       |          3 |
| [`barrier`](types/barrier.md)                                       | Graph, synchronization, and network | CREATE       |          1 |
| [`semaphore`](types/semaphore.md)                                   | Graph, synchronization, and network | CREATE       |          4 |
| [`rate_limit_bucket`](types/rate_limit_bucket.md)                   | Graph, synchronization, and network | CREATE       |          2 |
| [`cidr_network`](types/cidr_network.md)                             | Graph, synchronization, and network | SET          |          1 |
| [`url`](types/url.md)                                               | Graph, synchronization, and network | SET          |          2 |

## Nested typed values

Composite structures carry nested values as `{"type":"integer","value":42}`, so nested values keep their type.

## Common lifecycle

All keys support existence checks, type inspection, TTL, expiry, persistent TTL removal, deletion, and version tracking.
