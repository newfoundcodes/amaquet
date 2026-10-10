---
title: "Why Amaquet?"
description: "Compare Amaquet and Redis across protocol compatibility, typed data models, concurrency, persistence, security, operations, and deployment fit."
---

Redis is a mature data-structure server with a large client and operations ecosystem. Amaquet is a different product: a standalone, typed in-memory database written in Go with its own binary protocol, HTTP control plane, and optional append-only persistence. Amaquet is not a Redis-compatible server and should not be selected when Redis compatibility is the requirement.

The useful question is therefore not which product is universally better. It is which data model, protocol, topology, security boundary, and operational contract matches the application you are building.

## The short answer

Amaquet is a strong candidate when a new service needs an explicit typed data contract, a single-node in-memory data plane, and a control plane that is part of the same distribution.

- Choose Amaquet when the application can use the [Amaquet binary protocol](../protocol/overview.md), especially from Go, and should not inherit Redis command or wire compatibility as an architectural constraint.
- Choose Amaquet when values such as queues, streams, time series, vectors, indexes, locks, leases, semaphores, and other structures should have stable type identities and documented operations rather than being assembled from generic strings or hashes.
- Choose Amaquet when one process and one logical keyspace are an acceptable topology, with internal 256-shard concurrency, TTLs, versions, CAS, memory limits, eviction policy, and optional AOF persistence.
- Choose Amaquet when API keys, human members, roles, TOTP MFA, sessions, audit history, and the HTTP administration API should be delivered as one product boundary.
- Choose Redis when existing Redis clients, RESP compatibility, Redis commands, Redis modules, managed Redis offerings, replication, or Redis Cluster are central requirements.

## Comparison at a glance

The following comparison describes the current Amaquet implementation alongside the capabilities documented by Redis. It intentionally avoids throughput, latency, memory-efficiency, or reliability claims that would require a controlled benchmark or a deployment-specific evaluation.

| Aspect | Redis | Amaquet | Best fit |
| --- | --- | --- | --- |
| **Client and wire compatibility** | Clients communicate with Redis through [RESP](https://redis.io/docs/latest/develop/reference/protocol-spec/), with a broad ecosystem of libraries and tools. | Clients use Amaquet’s binary-framed protocol, JSON payloads, `HELLO` negotiation, request IDs, and the `amaquet://` or `amaquets://` URI schemes. | Redis for an existing Redis integration; Amaquet for a new, controlled client contract. |
| **Data model** | Redis provides native structures such as strings, hashes, lists, sets, sorted sets, streams, JSON, geospatial, probabilistic, time-series, and vector data. | Amaquet currently exposes 91 stable wire-level types with explicit type identity, typed nested values, `CREATE`, and type-specific `OP` operations. | Redis for Redis-native conventions; Amaquet when application values and operations should be explicit protocol types. |
| **Concurrency** | Redis provides command-level atomicity and optimistic locking through mechanisms such as `WATCH`, `MULTI`, and `EXEC`. | Amaquet tracks key versions, supports CAS, gives commands request IDs, supports cancellation, and documents `BATCH` as a round-trip optimization rather than a rollback transaction. | Redis when existing Redis transaction semantics are required; Amaquet when versioned typed mutations and cancellable requests fit the design. |
| **Memory governance** | Redis supports `maxmemory` and configurable eviction policies such as `noeviction`, LRU, LFU, random, and TTL-based policies. | Amaquet exposes approximate stored-data accounting, an optional memory ceiling, `noeviction`, LRU, LFU, and TTL-based eviction, plus adaptive compression for eligible values. | Either can fit a bounded cache; compare the exact accounting and eviction semantics with the workload. |
| **Persistence** | Redis supports RDB snapshots, AOF, no persistence, or combinations of persistence modes. | Amaquet keeps the active dataset in its own memory engine and optionally writes a CRC-protected `AMQTAOF2` journal with prepare/commit recovery, deterministic replay, checkpoints, compaction, and offline restore. | Redis for an established Redis persistence and recovery workflow; Amaquet for the repository-owned AOF and checkpoint contract. |
| **Topology** | Redis can run as a standalone instance and can scale out through [Redis Cluster](https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/), replication, and related deployment components. | Amaquet is single-node with one logical keyspace. Its 256 shards are internal concurrency partitions, not a cluster or replication protocol. | Redis for multi-node availability, failover, or horizontal scale; Amaquet when a single-node topology is intentional. |
| **Security and administration** | Redis provides ACLs and optional TLS; deployments commonly combine these with the surrounding Redis or platform operations model. | Amaquet has a separate HTTP administration listener, API keys, members, roles, editable RBAC, TOTP MFA, sessions, audit history, TLS enforcement, and optional Ed25519 protocol identity. | Redis for an existing Redis security stack; Amaquet when the data plane and control plane should ship together. |

The product boundaries matter more than the number of rows in this table. Redis and Amaquet both provide in-memory data structures, expiration, persistence choices, authentication, TLS, and memory policies, but the interfaces and operational semantics are not interchangeable.

## When Amaquet is the better fit

Amaquet is designed for applications that value an explicit server-side data contract and a focused deployment topology.

### Typed application state

Amaquet makes the public type identity part of the protocol and storage contract. Scalar values, collections, queues, messaging objects, time series, geospatial values, vectors, indexes, and synchronization structures are registered types rather than undocumented application conventions. A nested value also carries its own `type` and `value` envelope.

This is useful when several services or tools must agree on whether a value is an integer, decimal, queue entry, vector, timestamp, or structured object. The [data-type catalog](../data-types/index.md) and [operation reference](../data-types/operations.md) document the accepted arguments, results, defaults, persistence behavior, and failure behavior for each type.

Redis is often the better choice when the application already has a Redis data model built around RESP commands, strings, hashes, lists, streams, or Redis modules. Amaquet’s typed model is an advantage only when adopting a new protocol and type contract is acceptable.

### Go services with an owned protocol

Amaquet ships a supported Go client that negotiates protocol version 1, multiplexes requests by request ID, routes asynchronous event frames, supports TLS, and exposes typed helpers alongside a general `Command` method. The server and the client are maintained in the same repository, so a Go service can use the documented Amaquet contract without introducing Redis command compatibility into its own API boundary.

That does not mean every language can use Amaquet equally easily. A non-Go client must implement the Amaquet framing, negotiation, authentication, request, response, and event contracts. Redis is the simpler choice when the target language, framework, or platform already has a mature Redis client.

### Queues, messaging, and coordination

Amaquet includes first-class types for FIFO, blocking, delayed, reliable, priority, and ring-buffer queues, as well as streams, consumer groups, Pub/Sub, persistent topics, and event logs. It also includes synchronization and admission structures such as locks, leases, barriers, semaphores, and limit buckets.

The benefit is a documented object-level contract for the behavior the application needs. For example, the reliable-queue documentation covers claiming, acknowledgement, expiry, retry, and dead-letter behavior, while the synchronization types document their own state and operations.

Redis can be the better fit when workers already depend on Redis Lists, Streams, Pub/Sub, or a Redis-based coordination library. Amaquet’s structures are server-side Amaquet types; they are not Redis command aliases, and Amaquet’s single-node boundary should not be mistaken for a distributed lock or coordination service.

### A single-node data plane with a separate control plane

Amaquet deliberately separates the native TCP/TLS data-plane listener from the HTTP administration listener. Data commands use the Amaquet protocol; administration covers organization state, members, API keys, RBAC, persistence operations, runtime inspection, metrics, and audit history.

This fits a service that wants one deployable server with an explicit operational API instead of assembling the data server, identity model, administration endpoints, and audit storage from separate components. The administration state is stored in `data_dir/admin.json`; it is not mixed into the user keyspace or the data AOF.

### Controlled memory and persistence behavior

Amaquet’s engine supports TTLs, key versions, CAS, configurable memory admission, eviction policies, and transparent adaptive compression for eligible values. Optional AOF persistence uses resolved mutation data so time-dependent behavior such as expirations, generated stream IDs, and timestamps can be replayed deterministically. Verified checkpoints and `amaquet-restore` provide an explicit recovery path.

This is a good fit when the application wants to choose between cache-like operation and a journaled in-memory service while keeping the persistence format and recovery tooling within the Amaquet distribution. It is not a reason to treat Amaquet’s in-memory dataset as a replacement for independent backups or durable storage outside the service.

## When Redis is the better fit

Redis remains the more appropriate choice when compatibility, ecosystem, or distributed deployment requirements dominate.

### Existing Redis clients and commands

If an application already speaks RESP, uses Redis command names, depends on Redis modules, or relies on Redis-aware frameworks, switching to Amaquet is not a transparent replacement. Amaquet explicitly does not implement Redis RESP compatibility. Migration would require a client and data-model change, not only a new server address.

### Replication, failover, and cluster operation

Amaquet currently has no clustering, replication, automatic failover, or multi-node keyspace. Redis documents standalone operation alongside replication and Redis Cluster, which automatically distributes keys across nodes and uses replicas for failover scenarios.

Choose Redis when the service must continue through node failure using a Redis replication or Cluster topology, or when the dataset must scale across multiple Redis nodes. Choose Amaquet only when the single-node boundary is acceptable and is offset by the simpler topology or its typed/control-plane features.

### Redis ecosystem and managed operations

Redis has an established ecosystem of clients, command references, operational guides, monitoring integrations, modules, and managed offerings. Those integrations can be more valuable than Amaquet’s product-specific features when a team already operates Redis or needs a vendor-supported Redis deployment model.

Amaquet is a better fit when the team prefers a smaller, source-available Go service whose server, client, data types, protocol, tests, persistence, and documentation are in one repository. That is an ownership and integration choice, not a claim that Amaquet has the breadth or maturity of the Redis ecosystem.

## Decision guide by use case

Use the following as a starting decision, then validate the exact operations, recovery objectives, and deployment constraints for the application.

| Use case | Prefer Amaquet when… | Prefer Redis when… |
| --- | --- | --- |
| New Go service state | The service wants typed values, an owned Go client, versioned mutations, and a single-node server. | The service should use existing Redis libraries, RESP, or Redis commands. |
| Cache with TTL and bounded memory | Amaquet’s explicit memory limits, eviction choices, and optional compression match the service’s operational model. | The team already standardizes on Redis eviction, monitoring, or managed cache services. |
| Work queues | The application wants Amaquet queue types with documented retry, lease, blocking, delayed, or priority behavior. | Existing workers already use Redis Lists or Streams and their ecosystem is the main constraint. |
| Pub/Sub and event delivery | A service wants Amaquet event frames, typed payloads, persistent topics, or event logs within one server. | Existing applications and tooling already use Redis Pub/Sub or Streams. |
| Coordination | A single Amaquet node is the authority for locks, leases, barriers, semaphores, or admission controls. | Coordination must use an existing Redis deployment or a distributed Redis topology. |
| Durable in-memory service | Amaquet’s optional transactional AOF, deterministic replay, verified checkpoints, and offline restore match the recovery design. | Redis RDB/AOF workflows, replication, or managed durability are already established. |
| High availability and scale-out | The service does not require replication, cluster routing, or automatic failover. | Redis replication, Redis Cluster, or a managed Redis topology is required. |
| Existing production Redis | A migration is acceptable and the application benefits from Amaquet’s typed/control-plane model. | Compatibility and migration avoidance are more valuable than changing the server contract. |

## Important boundaries

Amaquet’s advantages are meaningful only when its boundaries are understood before deployment.

- Amaquet is not a drop-in Redis replacement. It does not implement RESP, Redis commands, Redis logical databases, Redis clustering, or replication.
- The 256 internal shards improve concurrent access within one process; they do not provide horizontal scaling or failover.
- The active dataset remains in memory. AOF is optional, and its `always`, `everysec`, and `no` fsync modes make different durability tradeoffs.
- `BATCH` reduces round trips but is not a rollback transaction. Do not infer Redis `MULTI`/`EXEC` semantics from it.
- Amaquet’s memory accounting is approximate stored-data accounting, not a hard RSS limit for the complete process.
- Explicit Amaquet index values are user-managed. Writing a document does not automatically update a separate index.

## Learn more

Read the [architecture](architecture.md), [memory and concurrency model](memory-model.md), [persistence and recovery](persistence.md), [security model](security.md), [protocol overview](../protocol/overview.md), and [implementation coverage](../reference/implementation-coverage.md) pages for Amaquet’s current behavior.

For the Redis side of the comparison, consult the official documentation for [data types](https://redis.io/docs/latest/develop/data-types/), [RESP](https://redis.io/docs/latest/develop/reference/protocol-spec/), [persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/), [ACLs](https://redis.io/docs/latest/operate/oss_and_stack/management/security/acl/), [TLS](https://redis.io/docs/latest/operate/oss_and_stack/management/security/encryption/), [eviction](https://redis.io/docs/latest/develop/reference/eviction/), and [Redis Cluster](https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/).
