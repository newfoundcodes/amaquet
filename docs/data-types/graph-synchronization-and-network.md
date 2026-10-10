---
title: "Graph, synchronization, and network"
---

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

See [Type construction and operations](operations.md#graph-synchronization-and-network-operations) for token/TTL rules, cancellation behavior, permit accounting, token-bucket refill, persistence exclusions, CIDR parsing, and URL accessors.

| Type                                              | Construction | Principal operations                 |
| ------------------------------------------------- | ------------ | ------------------------------------ |
| [`adjacency_set`](types/adjacency_set.md)         | CREATE       | ADD, REMOVE, NEIGHBORS               |
| [`atomic_counter`](types/atomic_counter.md)       | CREATE       | ADD, LOAD, CAS                       |
| [`lease`](types/lease.md)                         | CREATE       | ACQUIRE, RENEW, RELEASE, HELD        |
| [`lock`](types/lock.md)                           | CREATE       | TRY_LOCK, RENEW, UNLOCK              |
| [`barrier`](types/barrier.md)                     | CREATE       | WAIT                                 |
| [`semaphore`](types/semaphore.md)                 | CREATE       | ACQUIRE, TRY_ACQUIRE, RELEASE, STATS |
| [`rate_limit_bucket`](types/rate_limit_bucket.md) | CREATE       | ALLOW, REMAINING                     |
| [`cidr_network`](types/cidr_network.md)           | SET          | CONTAINS                             |
| [`url`](types/url.md)                             | SET          | HOST, SCHEME                         |
