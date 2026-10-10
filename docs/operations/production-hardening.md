---
title: "Production hardening"
---

Amaquet applies bounded-resource, fail-stop, and revocation rules for a single-node production deployment.

## Memory governance

`limits.max_memory_bytes` is the storage-engine data budget. New and replacement values use their **net** stored-size change. A same-size replacement does not require a second full value in the engine accounting model.

The Go runtime also receives a soft process memory limit with headroom when `max_memory_bytes` is non-zero. `INFO` exposes aggregate engine memory, `MEMORY` exposes approximate per-key storage, and `/metrics` adds Go heap allocation. The process limit is a runtime guard, not a byte-exact RSS guarantee because TLS, kernel socket buffers, mapped libraries, and non-Go allocations are outside the engine counter.

Supported eviction policies are `noeviction`, `allkeys-lru`, `allkeys-lfu`, and `volatile-ttl`. Eviction uses bounded sampling. Memory-reducing operations remain available when the database is at its limit.

Composite mutations reserve capacity before state becomes visible. The engine checks the exact stored-size delta when the mutation metadata is committed.

## Expiration scheduling

Each expiring key has one indexed expiration-heap node. `EXPIRE` updates that node. `PERSIST` removes it. Repeated TTL changes therefore do not create unbounded stale heap entries.

## Network resource limits

The protocol server enforces:

- maximum connections;
- maximum in-flight requests for the server;
- maximum in-flight requests for one connection;
- maximum frame payload size;
- maximum aggregate in-flight payload bytes;
- duplicate active request-ID rejection;
- configurable command execution timeout;
- read and write deadlines;
- authentication failure throttling.

`limits.command_timeout_ms` is the default server-side command deadline. A cancellation frame can cancel a compatible active request before that deadline.

## Large binary values

A single Amaquet frame remains bounded. Large `binary_string` values use the chunked upload commands.

The upload path has these safeguards:

- upload data is spooled to a private temporary file before commit;
- zero-byte chunks are rejected;
- overlapping chunks are rejected;
- one upload can contain at most 65,536 accepted spans;
- span lookup uses ordered binary search instead of sorting the complete span list after every chunk;
- total pending upload bytes are bounded;
- uploads expire after `limits.upload_ttl_ms` of inactivity;
- connection-scoped uploads require an exact owner match;
- abandoned connection-scoped uploads are removed when the connection closes;
- incomplete replayed uploads are removed before traffic is accepted.

At `BLOB_COMMIT`, Amaquet reads the temporary file into bounded 1 MiB in-memory blocks. It does **not** allocate one contiguous buffer equal to the complete object size. `BLOB_READ` returns a bounded range and is the preferred read path for large values. A normal `GET` returns metadata instead of materializing a large chunked value into one protocol response.

The current maximum declared blob size is 1 GiB. `BLOB_READ.length` is limited to 32 MiB per request.

## Credential revocation

API-key revocation is applied to new and existing access paths.

- New authentication with the key fails.
- Existing server-side sessions are revalidated against the actor state.
- Registered revocation hooks remove matching HTTP cookie sessions.
- Matching active Amaquet connections are closed by the server process.
- Every protocol command also revalidates its authenticated actor.
- Active Pub/Sub delivery revalidates the actor before each event.

Member disable/delete, member credential rotation, and MFA activation use the same actor-revocation mechanism.

## Pub/Sub write deadlines

Normal responses and asynchronous Pub/Sub event frames refresh the socket write deadline before writing. An idle subscription therefore does not inherit an expired deadline from its original subscribe response.

## Persistence fail-stop behavior

AOF v2 uses `prepare`, `commit`, and `abort` records with CRC validation. Mutations are serialized through a mutation barrier when AOF is active.

If the journal commit fails after an in-memory mutation has already been applied, persistence health changes to failed. New durable writes are then rejected. `/api/ready` reports the node as not ready. This fail-stop rule prevents the server from continuing to accept writes after known durable-state divergence.

## Compact checkpoints and restore

An online checkpoint is not a blind file copy. Amaquet resolves committed transactions, removes obsolete histories where the operation semantics permit it, writes a new AMQTAOF2 recovery image, fsyncs it, and replays it for verification before publishing the destination.

Blob compaction retains only the latest committed upload sequence for a key. Incomplete and aborted uploads are omitted.

Use `amaquet-restore` while the server is stopped to install a checkpoint. The tool validates the source, writes a temporary destination, fsyncs it, verifies it again, and atomically swaps it into place with rollback protection.

## Control-plane bounds

The admin server provides:

- bounded HTTP cookie sessions;
- expired-session cleanup;
- session invalidation on actor revocation;
- authentication failure throttling;
- bounded failure-tracking maps;
- optional authentication on `/metrics`;
- crash-safe admin-state writes;
- batched persistence of API-key `last_used_at` metadata;
- explicit admin-state schema versions and migrations.

HTTP sessions use `HttpOnly`, `SameSite=Strict` cookies. The cookie is also `Secure` when HTTPS is active.

## Human member identities

Team members are not only metadata. An administrator can create a one-time invitation for a member. The member accepts it once and receives a member access token. Member tokens inherit the member's current RBAC role and active/disabled state.

Members can enable TOTP MFA. After MFA is active, a member access token alone cannot create a new interactive session; the six-digit TOTP code is also required. Enabling MFA revokes existing actor sessions/connections so the next login uses the new factor.

API keys remain the preferred machine-to-machine credential.

## Bootstrap credential

If no bootstrap token is supplied, startup creates one under `data_dir/bootstrap-token` with mode `0600` and logs only the file path. The token value is not written to the application log.

The file is reused across restarts until bootstrap is disabled. When bootstrap creates the first administrator API key, bootstrap authentication is permanently disabled in admin state.

API configuration writes never copy an environment-provided bootstrap token into the JSON configuration file.

## Docker defaults

The supplied development Compose file publishes the Amaquet protocol and administration API ports on `127.0.0.1`. Remote production deployments should use `amaquets://` and admin HTTPS.

## Chaos and soak testing

Use:

```bash
make chaos-smoke
make chaos
```

The suite covers kill/restart recovery, abandoned uploads, memory eviction, connection churn, TTL churn, and mixed sustained workloads. Scheduled CI also runs longer chaos, fuzz, and benchmark jobs.
