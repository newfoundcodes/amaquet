---
title: "Collections"
---

Collections are synchronized mutable structures manipulated with `OP`.

See [Type construction and operations](operations.md#collection-operations) for exact arguments, results, edge cases, and implementation costs. In particular, list front operations use slices, deques use a circular buffer, ring buffers overwrite the oldest entry at capacity, and sorted-set range reads sort the current entries by score.

| Type                                  | Construction | Principal operations                                    |
| ------------------------------------- | ------------ | ------------------------------------------------------- |
| [`list`](types/list.md)               | CREATE       | PUSH_FRONT, PUSH_BACK, POP_FRONT, POP_BACK, RANGE, LEN  |
| [`array`](types/array.md)             | CREATE       | APPEND, SET, GET, VALUES, LEN                           |
| [`deque`](types/deque.md)             | CREATE       | PUSH_FRONT, PUSH_BACK, POP_FRONT, POP_BACK, VALUES, LEN |
| [`ring_buffer`](types/ring_buffer.md) | CREATE       | PUSH, VALUES, LEN                                       |
| [`tuple`](types/tuple.md)             | CREATE       | GET, VALUES, LEN                                        |
| [`set`](types/set.md)                 | CREATE       | ADD, REMOVE, CONTAINS, MEMBERS, LEN                     |
| [`sorted_set`](types/sorted_set.md)   | CREATE       | ADD, REMOVE, SCORE, RANGE                               |
| [`ordered_set`](types/ordered_set.md) | CREATE       | ADD, REMOVE, VALUES                                     |
| [`multiset`](types/multiset.md)       | CREATE       | ADD, REMOVE, COUNT                                      |
| [`hashmap`](types/hashmap.md)         | CREATE       | SET, GET, DELETE, KEYS                                  |
| [`ordered_map`](types/ordered_map.md) | CREATE       | SET, GET, DELETE, ENTRIES                               |
| [`multimap`](types/multimap.md)       | CREATE       | ADD, GET, REMOVE                                        |
