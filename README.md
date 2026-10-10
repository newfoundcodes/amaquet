# Amaquet

[![CI](https://github.com/newfoundcodes/amaquet/actions/workflows/ci.yml/badge.svg)](https://github.com/newfoundcodes/amaquet/actions/workflows/ci.yml) [![Soak](https://github.com/newfoundcodes/amaquet/actions/workflows/soak.yml/badge.svg)](https://github.com/newfoundcodes/amaquet/actions/workflows/soak.yml)

Amaquet is a standalone, Redis-like in-memory database written in Go. It does **not** use Redis or another database as its storage engine. It provides its own `amaquet://` binary-framed protocol, a strongly typed data model, sharded concurrent storage, TTL/versioning, an optional append-only journal, API-key authentication, RBAC, and an HTTP administration API.

## Main components

- **Database engine:** Go, 256-way sharded keyspace, scheduled TTL expiration, optimistic versions/CAS, cursor scans, approximate max-memory governance, and configurable eviction.
- **Amaquet protocol:** custom multiplexed `Amaquet` binary frame format over TCP with request IDs and cancellation. `amaquet://` is plain TCP and `amaquets://` is TLS.
- **Typed values:** 91 data types are registered as stable wire identifiers. See the [data-type catalog](docs/data-types/index.md) and [complete operation reference](docs/data-types/operations.md).
- **Advanced structures:** queues, streams, consumer groups, Pub/Sub, persistent topics, event logs, probabilistic structures, time series, geospatial indexes, vectors/HNSW, search indexes, and synchronization primitives.
- **Security:** API keys, human member identities, one-time invitations, optional member TOTP MFA, admin/auditor/developer roles, configurable RBAC, one-time bootstrap, browser `HttpOnly` sessions, immediate actor revocation, TLS enforcement for remote authenticated listeners, and optional Ed25519 identity signatures.
- **Control plane:** Go HTTP API for organization, identity, RBAC, API keys, runtime inspection, persistence operations, and audit history.
- **Administration state:** organization, members, RBAC, API-key hashes, and audit history are stored in a local JSON control-plane file. Database values remain in Amaquet's own engine.
- **Optional AOF:** transactional `AMQTAOF2` write-ahead journal with CRC protection, deterministic replay, legacy migration, verified checkpoints, compaction, and persistence-health reporting.

## Repository layout

```text
cmd/amaquet/              Amaquet server
cmd/amaquet-cli/          command-line Amaquet client
cmd/amaquet-keygen/       Ed25519 identity key generator
cmd/amaquet-restore/      Verified offline checkpoint restore tool
internal/core/         sharded keyspace and stable data-type registry
internal/types/        concrete data structures
internal/protocol/     Amaquet frame format and URI parser
internal/server/       TCP server, dispatcher, factories
internal/admin/        RBAC/control-plane state and HTTP API
internal/security/     Ed25519 server identity
internal/persistence/  optional append-only journal
pkg/amaquet/              reusable Go client
docs/                  Astro/Starlight documentation site and content
tests/                 Go integration, per-type fuzzers, developer scenarios, security/AOF fuzzing
```

## Documentation and Bash verification

The repository contains a complete Astro/Starlight site and one deterministic Bash test plus one Bash fuzzer for every one of the 91 stable wire types.

```bash
# Complete per-type semantic suite
./tests/bash/run_all.sh

# Per-type fuzzing
AMAQUET_FUZZ_ITERATIONS=100 ./tests/bash/run_fuzz.sh

# Raw Amaquet frame fuzzing
AMAQUET_FRAME_FUZZ_ITERATIONS=1000 ./tests/bash/run_protocol_fuzz.sh

# Cross-feature developer workflows
AMAQUET_SCENARIO_ITERATIONS=10 ./tests/bash/run_scenarios.sh

# Restart/auth/AOF fuzzing
./tests/bash/run_isolated_fuzz.sh

# Full black-box developer fuzz suite
./tests/bash/run_developer_fuzz.sh

# Native Go coverage-guided fuzz targets
AMAQUET_GO_FUZZ_TIME=10s ./scripts/run_go_fuzz.sh

# Dependency-free documentation consistency check
python3 docs/scripts/check_docs.py

# Build/serve the Astro documentation site
make docs-build
make docs-serve
```

See [Testing and fuzzing](docs/development/testing-fuzzing.md) and the [documentation home](docs/index.md).

## Requirements

- Go 1.23 or newer
- Node.js 22.12 or newer for the documentation site
- npm

## Build from source

```bash
go test ./...
go build -o bin/amaquet ./cmd/amaquet
go build -o bin/amaquet-cli ./cmd/amaquet-cli
go build -o bin/amaquet-keygen ./cmd/amaquet-keygen
go build -o bin/amaquet-restore ./cmd/amaquet-restore
```

Or use:

```bash
make build
```

The default build compiles the four Go binaries. Documentation dependencies and builds remain available through `make docs-install` and `make docs-build`.

## Initialize

Create a default configuration with a random bootstrap administrator token:

```bash
./bin/amaquet --init-config --config ./amaquet.json
```

The token is printed once and is also written to the configuration file. Protect that file.

Generate an optional Ed25519 server identity:

```bash
mkdir -p keys
./bin/amaquet-keygen \
  --public ./keys/amaquet-public.pem \
  --private ./keys/amaquet-private.pem
```

Set the generated paths under `security.public_key_file` and `security.private_key_file`.

## Run

```bash
./bin/amaquet --config ./amaquet.json
```

Defaults:

- Amaquet protocol: `127.0.0.1:13378`
- Admin API: `127.0.0.1:13379`
- URI: `amaquet://127.0.0.1:13378`
- TLS URI: `amaquets://example.com:13378`

The administration listener exposes `/api/*` and `/metrics`; it does not serve a user interface. Authenticate API requests with the bootstrap token, an API key, a member token, or a server-side session as described in the [Admin API reference](docs/reference/admin-api.md).

## First API key

The bootstrap token initializes the control plane. After it is used to create the first `admin` API key, bootstrap authentication is permanently disabled in the persisted administration state. Amaquet stores only SHA-256 hashes of generated high-entropy API-key secrets and displays each raw secret once.

Roles are:

- `admin`: unrestricted by default.
- `auditor`: read-only data and management visibility by default.
- `developer`: data read/write plus selected management visibility by default.

RBAC permissions are editable through the administration API.

## Go client

```go
ctx := context.Background()
client, err := amaquet.DialWithOptions(ctx,
    "amaquet://127.0.0.1:13378",
    amaquet.DialOptions{APIKey: os.Getenv("AMAQUET_API_KEY")})
if err != nil { /* handle */ }
defer client.Close()

err = client.Set(ctx, "counter", "integer", int64(41), 0)

var counter int64
err = client.Op(ctx, "counter", "ADD", map[string]any{
    "delta": 1,
}, &counter)
```

Composite values are created with `CREATE` and manipulated with `OP`:

```go
err = client.Create(ctx, "jobs", "fifo_queue", nil)

job := map[string]any{
    "type": "json",
    "value": map[string]any{"id": "job-1"},
}
err = client.Op(ctx, "jobs", "ENQUEUE", map[string]any{
    "value": job,
}, nil)
```

## CLI

```bash
./bin/amaquet-cli \
  -uri 'amaquet://127.0.0.1:13378' \
  -api-key "$AMAQUET_API_KEY" \
  SET '{"key":"answer","type":"integer","value":42}'

./bin/amaquet-cli \
  -uri 'amaquet://127.0.0.1:13378' \
  -api-key "$AMAQUET_API_KEY" \
  GET '{"key":"answer"}'
```

## Common commands

The Amaquet command payload supports these keyspace commands:

| Command | Purpose |
|---|---|
| `PING` | Liveness check |
| `INFO` | Server name, protocol version, uptime, key/type counts |
| `TYPES` | Return all registered data-type names |
| `SET` | Create/replace a directly encoded typed value |
| `CREATE` | Create a composite or specialized value |
| `GET` | Read a key, its type, value snapshot, and version |
| `DEL` | Delete keys |
| `EXISTS` | Test key existence |
| `TYPE` | Read the stable type name |
| `TTL` | Read remaining TTL in milliseconds |
| `EXPIRE` | Apply a TTL |
| `PERSIST` | Remove a TTL |
| `KEYS` | Bounded administrative key listing |
| `SCAN` | Cursor-based key iteration |
| `CAS` | Optimistic versioned replacement |
| `BATCH` | Reduce round trips for a list of requests |
| `BLOB_BEGIN` / `BLOB_CHUNK` / `BLOB_COMMIT` | Upload a large binary value in bounded frames |
| `OP` | Apply a type-specific operation |

Every `OP` request has:

```json
{
  "command": "OP",
  "args": {
    "key": "my-key",
    "operation": "...",
    "args": {}
  }
}
```

Nested values use an explicit wire value:

```json
{
  "type": "utf8_string",
  "value": "hello"
}
```

## Pub/Sub

Pub/Sub uses dedicated protocol opcodes so messages arrive as asynchronous Amaquet event frames rather than normal command responses.

```go
sub, err := client.Subscribe(ctx, "bus", "updates", 64)
if err != nil { /* handle */ }
defer sub.Close()

for message := range sub.C {
    fmt.Println(message.Channel, message.Payload)
}
```

`persistent_topic` is separate from Pub/Sub. It stores ordered topic messages in memory and can be included in the AOF operation journal.

## TLS and protocol identity

TLS is enabled by setting:

```json
{
  "protocol": {
    "tls": true,
    "tls_cert_file": "./certs/server.crt",
    "tls_key_file": "./certs/server.key"
  }
}
```

Clients then use `amaquets://...`.

Amaquet can also load an Ed25519 identity. `HELLO` accepts a client nonce and returns the public key, fingerprint, and nonce signature. This is separate from TLS and can be used for server pinning or application-level identity verification.

## Optional append-only persistence

Amaquet is an in-memory database. Persistence is optional and does not replace the in-memory engine.

```json
{
  "persistence": {
    "aof_enabled": true,
    "aof_path": "./data/amaquet.aof",
    "fsync": "everysec"
  }
}
```

The AOF stores successful durable mutations with a length and CRC32 checksum. On startup, complete valid records are replayed before the network listener starts. A partial last record is ignored. Live coordination state such as Pub/Sub subscriptions, leases, locks, barriers, semaphores, and token-bucket state is deliberately not journaled.

Existing pre-Amaquet v1 and v2 journals are recognized and atomically rewritten with the current `AMQTAOF2` header when opened. See the [deployment migration procedure](docs/operations/deployment.md#migrating-an-existing-installation-to-amaquet) before upgrading a persisted installation; client URI schemes, wire magic, environment variables, binary names, metrics, and automation headers changed with the rename.

## Docker

```bash
export AMAQUET_BOOTSTRAP_TOKEN='replace-with-a-long-random-secret'
docker compose up --build
```

Ports `13378` and `13379` are exposed. Database/control-plane local state is kept in the `amaquet-data` volume.

## Verification

The repository includes unit and integration tests for the engine, protocol, identity keys, control-plane state, all major data-structure families, and an end-to-end authenticated Amaquet connection including server nonce verification and Pub/Sub.

```bash
go test ./...
go test -race ./...
go vet ./...
```

## Design boundaries

Amaquet deliberately keeps several concepts separate:

- **Data types** are values stored under keys.
- **Indexes** are explicit in-memory index objects; automatic query planning is not implied.
- **Pub/Sub** is transient messaging; `persistent_topic` and `event_log` are retained in-memory structures.
- **Synchronization objects** coordinate live processes and are not restored from AOF because a restarted process must not recreate stale waiters or leases.
- **Admin state** is control-plane state, not user database data.

See the [architecture](docs/concepts/architecture.md), [protocol reference](docs/protocol/overview.md), and [data-type catalog](docs/data-types/index.md).

### Adaptive in-memory compression

Amaquet includes transparent adaptive compression for large payload-oriented values. The default `auto` mode uses low-latency LZ4-block and RLE candidates and retains a compressed representation only when it exceeds the configured saving threshold. `deflate-fast` is also available when a higher compression ratio is worth additional CPU. Use `MEMORY` for per-key physical storage information and `INFO` for aggregate compression statistics. See `docs/concepts/compression.md`.

## License

Amaquet is licensed under the GNU Affero General Public License, version 3 (AGPLv3). See [LICENSE](LICENSE) for the complete license text.
