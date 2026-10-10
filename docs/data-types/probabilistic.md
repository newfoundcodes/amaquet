---
title: "Probabilistic"
---

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

See [Type construction and operations](operations.md#probabilistic-operations) for parameter constraints, hashing, saturation, candidate bounds, approximation behavior, and false-positive/estimation semantics. These structures do not provide exact membership or exact frequency guarantees.

| Type                                                      | Construction | Principal operations  |
| --------------------------------------------------------- | ------------ | --------------------- |
| [`hyperloglog`](types/hyperloglog.md)                     | CREATE       | ADD, COUNT            |
| [`bloom_filter`](types/bloom_filter.md)                   | CREATE       | ADD, CONTAINS         |
| [`counting_bloom_filter`](types/counting_bloom_filter.md) | CREATE       | ADD, REMOVE, CONTAINS |
| [`cuckoo_filter`](types/cuckoo_filter.md)                 | CREATE       | ADD, REMOVE, CONTAINS |
| [`count_min_sketch`](types/count_min_sketch.md)           | CREATE       | ADD, ESTIMATE         |
| [`top_k`](types/top_k.md)                                 | CREATE       | ADD, ITEMS            |
| [`t_digest`](types/t_digest.md)                           | CREATE       | ADD, QUANTILE         |
