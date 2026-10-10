---
title: "Configuration reference"
description: "Reference every Amaquet JSON configuration field, including defaults, accepted values, validation rules, security implications, and operational effects."
---

This table documents every supported JSON configuration field, its accepted type, default value, and operational effect.

| Path                                 | Type    | Default              | Notes                                                                               |
| ------------------------------------ | ------- | -------------------- | ----------------------------------------------------------------------------------- |
| `protocol.host`                      | string  | `127.0.0.1`          | Amaquet bind address                                                                |
| `protocol.port`                      | integer | `13378`              | 1–65535                                                                             |
| `protocol.tls`                       | boolean | `false`              | enables Amaquet TLS                                                                 |
| `protocol.tls_cert_file`             | string  | empty                | required with Amaquet TLS                                                           |
| `protocol.tls_key_file`              | string  | empty                | required with Amaquet TLS                                                           |
| `protocol.require_auth`              | boolean | `true`               | require Amaquet authentication                                                      |
| `protocol.allow_insecure_auth`       | boolean | `false`              | permit authenticated plaintext on non-loopback only when explicitly set             |
| `protocol.read_timeout_ms`           | integer | `120000`             | frame/idle read deadline; `0` disables                                              |
| `protocol.write_timeout_ms`          | integer | `30000`              | response/event write deadline; `0` disables                                         |
| `admin.host`                         | string  | `127.0.0.1`          | admin bind address                                                                  |
| `admin.port`                         | integer | `13379`              | 1–65535                                                                             |
| `admin.tls`                          | boolean | `false`              | enable HTTPS                                                                        |
| `admin.tls_cert_file`                | string  | empty                | required with admin TLS                                                             |
| `admin.tls_key_file`                 | string  | empty                | required with admin TLS                                                             |
| `admin.allow_insecure`               | boolean | `false`              | permit non-loopback HTTP only when explicitly set                                   |
| `admin.metrics_require_auth`         | boolean | `true`               | require RBAC authentication for `/metrics`                                          |
| `admin.max_sessions`                 | integer | `10000`              | maximum stored HTTP cookie sessions                                                 |
| `admin.auth_failures_per_minute`     | integer | `20`                 | failed-login throttle per source address                                            |
| `security.bootstrap_token`           | string  | empty                | supplied bootstrap secret; if empty, startup uses a protected file under `data_dir` |
| `security.public_key_file`           | string  | empty                | Ed25519 public identity key                                                         |
| `security.private_key_file`          | string  | empty                | Ed25519 private identity key                                                        |
| `persistence.aof_enabled`            | boolean | `false`              | transactional AOF journal                                                           |
| `persistence.aof_path`               | string  | `./data/amaquet.aof` | AOF path                                                                            |
| `persistence.fsync`                  | enum    | `everysec`           | `always`, `everysec`, or `no`                                                       |
| `limits.max_connections`             | integer | `10000`              | accepted Amaquet connections                                                        |
| `limits.max_payload_bytes`           | integer | 64 MiB               | per-frame payload; maximum 64 MiB                                                   |
| `limits.max_memory_bytes`            | integer | `0`                  | engine data budget; `0` means unlimited                                             |
| `limits.eviction_policy`             | enum    | `noeviction`         | `noeviction`, `allkeys-lru`, `allkeys-lfu`, `volatile-ttl`                          |
| `limits.max_inflight_requests`       | integer | `4096`               | server-wide active command cap                                                      |
| `limits.max_inflight_per_connection` | integer | `128`                | active command cap for one connection                                               |
| `limits.max_inflight_payload_bytes`  | integer | 256 MiB              | aggregate inbound frame budget                                                      |
| `limits.max_pending_upload_bytes`    | integer | 1 GiB                | total declared bytes for active blob uploads                                        |
| `limits.upload_ttl_ms`               | integer | `600000`             | upload inactivity TTL; minimum 1000 ms                                              |
| `limits.command_timeout_ms`          | integer | `30000`              | server-side command execution deadline                                              |
| `compression.enabled`                | boolean | `true`               | adaptive compression for supported contiguous value types                           |
| `compression.algorithm`              | enum    | `auto`               | `auto`, `lz4`, `rle`, `deflate-fast`, `none`                                        |
| `compression.min_bytes`              | integer | `1024`               | smaller values remain native                                                        |
| `compression.min_savings_percent`    | number  | `8`                  | required saving before a compressed encoding is retained                            |
| `data_dir`                           | string  | `./data`             | admin state, upload spools, backups, and runtime security files                     |

The table is a compact inventory. The sections below explain what each key changes, when the setting is read, and what to consider before changing it.

## How configuration is loaded

Amaquet starts from built-in defaults, reads the JSON file named by `-config`, overlays the supported `AMAQUET_*` environment variables, validates the resulting configuration, and only then starts either listener. The default path is `./amaquet.json`; a missing file is allowed and therefore produces a defaults-plus-environment configuration. A malformed file, invalid value, missing TLS material, or unsafe non-loopback listener stops startup.

JSON is the durable configuration source. Environment variables are process-start overrides and take precedence over values read from JSON for the fields they support. They are not written back by `amaquet --init-config` or by the administration configuration endpoint. Relative paths are interpreted by the server process from its current working directory, so service units should set an explicit `WorkingDirectory` or use absolute paths.

`amaquet --init-config` writes the default structure and a generated bootstrap token, then exits. It does not start the server. Normal startup may create `data_dir` and, when no bootstrap token is configured, create `data_dir/bootstrap-token` with mode `0600`.

## Protocol listener

The `protocol` object controls the native Amaquet binary protocol. Applications connect to this listener with `amaquet://` for plaintext or `amaquets://` for TLS. It is the data-plane endpoint used by `amaquet-cli` and the Go client; it is separate from the HTTP administration listener.

### `protocol.host`

The local address on which the native protocol socket listens. `127.0.0.1` keeps the listener local to the machine and is the safest development default. Use a specific private interface in a multi-interface deployment when possible. Binding `0.0.0.0` exposes the listener on every IPv4 interface and should be paired with TLS, authentication, firewall rules, and a connection limit.

### `protocol.port`

The TCP port for the native protocol. It must be between `1` and `65535`; the default is `13378`. Changing this value also changes the URI passed to clients. The port must not collide with the admin listener or another process.

### `protocol.tls`

Enables TLS for the native protocol. When enabled, clients must use an `amaquets://` URI and validate the configured certificate in the normal way. TLS protects credentials and data in transit; it does not replace protocol authentication. TLS startup fails unless both certificate and key paths are supplied.

### `protocol.tls_cert_file`

Filesystem path to the certificate presented by the native protocol listener. The file must be readable by the Amaquet process and contain a certificate whose names match the hostnames clients use. The setting is only used when `protocol.tls` is enabled, although the environment variable `AMAQUET_TLS_CERT` also enables TLS automatically.

### `protocol.tls_key_file`

Filesystem path to the private key corresponding to `protocol.tls_cert_file`. Protect this file with filesystem permissions and do not place it in a public repository or image layer. Amaquet refuses to start TLS without both certificate and key files.

### `protocol.require_auth`

Controls whether new native-protocol connections must authenticate before issuing protected commands. The default is `true`. Authentication accepts a bootstrap credential, API key, or eligible member token through the protocol `AUTH` flow. Setting this to `false` makes new connections anonymous administrative actors, so use it only on an isolated trusted loopback or test deployment.

### `protocol.allow_insecure_auth`

Explicitly permits authenticated plaintext protocol traffic when the listener is not loopback. It does not disable authentication and does not enable TLS. This is a narrowly scoped development escape hatch for a protected network; production deployments should leave it `false` and use TLS instead. Validation rejects a non-loopback authenticated plaintext listener unless this key is enabled.

### `protocol.read_timeout_ms`

Sets the read deadline for an incomplete frame or an idle connection, in milliseconds. The default is `120000` (two minutes). A value of `0` disables this deadline. Increase it only for clients that legitimately stream or assemble requests slowly; disabling it allows abandoned connections to consume resources indefinitely.

### `protocol.write_timeout_ms`

Sets the response and event write deadline, in milliseconds. The default is `30000` (30 seconds); `0` disables the deadline. This protects the server from clients that stop reading responses, including event or Pub/Sub delivery. A value that is too small can disconnect slow but healthy clients.

## Administration listener

The `admin` object controls the HTTP control-plane listener. It serves `/api/*` and `/metrics`; it does not serve the documentation site or a browser UI. Keep it on loopback or a private management network, and expose it remotely only with TLS or an explicitly controlled reverse proxy.

### `admin.host`

The local address for the HTTP administration server. The default `127.0.0.1` prevents remote access. A non-loopback address requires `admin.tls` or the explicit `admin.allow_insecure` override during validation.

### `admin.port`

The HTTP administration port. It must be between `1` and `65535`; the default is `13379`. Health checks, metrics scrapers, and administration clients must use this port, not the native protocol port.

### `admin.tls`

Enables HTTPS for the administration server. When enabled, the admin listener requires its own certificate and key settings. Keep the admin certificate's names aligned with the management hostname. HTTPS protects bootstrap credentials, API keys, sessions, and configuration operations in transit.

### `admin.tls_cert_file`

Path to the certificate used by the HTTPS administration listener. It is independent of `protocol.tls_cert_file`, allowing different trust and hostname policies for data-plane and control-plane traffic.

### `admin.tls_key_file`

Path to the private key paired with `admin.tls_cert_file`. The Amaquet process must be able to read it at startup; protect it as a secret. `admin.tls` fails validation unless both admin TLS paths are present.

### `admin.allow_insecure`

Allows plaintext admin HTTP on a non-loopback address. This setting is intended for a deliberately isolated development network or a reverse-proxy arrangement where the proxy provides the required protection. It does not authenticate requests by itself. Leave it `false` for production and use HTTPS.

### `admin.metrics_require_auth`

Controls whether `/metrics` requires administration authentication. The default is `true`, keeping operational data and labels behind the same access boundary as other admin endpoints. Set it to `false` only when the listener is already isolated and the metrics exposure is intentionally public to the monitoring network.

### `admin.max_sessions`

Caps the number of server-side HTTP cookie sessions kept in memory. The default is `10000`. Sessions are created by the admin login/session flow and are distinct from native protocol connections and API keys. Lower this value on a constrained control-plane deployment; increasing it consumes more memory and does not increase the number of API keys.

### `admin.auth_failures_per_minute`

Limits failed authentication attempts per source address per minute for the admin and native authentication hardening paths. The default is `20`; it must be positive. This is a throttle, not a replacement for TLS, network access control, or strong credentials.

## Security and identity

The `security` object contains bootstrap and optional protocol-identity material. Bootstrap credentials authorize the first administration actions; Ed25519 identity keys let clients verify that they reached the expected server identity after protocol negotiation. Identity keys are not TLS certificates and do not encrypt traffic.

### `security.bootstrap_token`

The initial administrative credential. `amaquet --init-config` generates one and writes it into the JSON file. If it is empty during normal startup, Amaquet reads or creates `data_dir/bootstrap-token` with restrictive permissions and logs the protected file path rather than the secret itself. Use the bootstrap credential to create a permanent API key, then stop using it; persisted administration state permanently disables bootstrap authentication after the first admin API key is created.

Do not commit this value, pass it in shell history, or put it in a public Compose file. `AMAQUET_BOOTSTRAP_TOKEN` overrides it at runtime without copying the secret into the JSON configuration.

### `security.public_key_file`

Path to the Ed25519 public identity key generated by `amaquet-keygen`. It is loaded together with `security.private_key_file` and its fingerprint is advertised during protocol negotiation. Supplying only one identity path is not enough to enable identity loading.

### `security.private_key_file`

Path to the matching Ed25519 private identity key. Restrict access to the Amaquet service account. A client should verify the advertised fingerprint/signature when it needs application-level server identity; use TLS as well when confidentiality and certificate validation are required.

## Persistence

Persistence is optional. The active keyspace remains in memory in both modes; AOF provides recovery after restart, not a disk-backed replacement for the engine. Keep the journal and administration state under protected storage and back them up independently.

### `persistence.aof_enabled`

Enables transactional append-only journaling. On startup Amaquet replays committed mutations before opening listeners. When `false`, the data keyspace starts empty after a restart, although administrative state can still remain in `data_dir/admin.json`. Enabling AOF adds write and storage overhead, so select the durability mode with the workload's recovery objective in mind.

### `persistence.aof_path`

Path to the AOF journal. The parent directory must be writable by the service account. Use an explicit path on persistent storage in production rather than relying on `./data` relative to an unpredictable working directory. Checkpoint and restore workflows should use separate backup copies rather than replacing the live journal by hand.

### `persistence.fsync`

Selects how aggressively journal data is flushed to durable storage:

- `always` flushes and fsyncs each journal phase, maximizing durability at the cost of write latency.
- `everysec` flushes approximately once per second and is the normal latency/durability balance.
- `no` relies on buffered and operating-system writeback and provides the weakest crash-durability guarantee.

All modes retain transactional recovery semantics. A complete record with a bad CRC is treated as corruption; an incomplete final tail can be discarded as an interrupted write.

## Resource limits

The `limits` object protects the process from connection floods, oversized frames, concurrent work, blob-upload reservations, and unbounded data growth. These limits are independent: raising one does not automatically raise the others.

### `limits.max_connections`

Maximum number of accepted native protocol connections. The default is `10000` and the value must be positive. Size it with the operating system's file-descriptor limit, expected client pool count, TLS overhead, and admin connections in mind.

### `limits.max_payload_bytes`

Maximum payload in one protocol frame. The default is `67108864` bytes (64 MiB), which is also the hard upper bound. Lower it to reduce per-request memory pressure. Values larger than one frame should use the chunked blob commands where supported; this limit does not define the maximum logical value size for every data type.

### `limits.max_memory_bytes`

Approximate memory budget for data held by the Amaquet engine. `0` disables the configured engine budget. It is not a process-RSS cap: Go runtime metadata, network buffers, indexes, temporary decoding buffers, AOF state, and the admin server consume additional memory. Leave headroom between this budget and the container or host memory limit.

### `limits.eviction_policy`

Selects what happens when a mutating operation would exceed `max_memory_bytes`:

- `noeviction` rejects the allocating write.
- `allkeys-lru` removes the least-recently-used sampled candidate, whether or not it has a TTL.
- `allkeys-lfu` removes a low-frequency sampled candidate.
- `volatile-ttl` considers only keys with expiration and prefers the key nearest expiry; if no eligible key exists, the write is rejected.

Eviction is approximate and sampled rather than a perfect global ordering. Destructive or memory-reducing commands such as `DEL` remain available when the engine is over budget.

### `limits.max_inflight_requests`

Server-wide cap on active command executions. The default is `4096`. It limits total concurrent work across all protocol connections and helps prevent a small number of clients from exhausting worker capacity.

### `limits.max_inflight_per_connection`

Per-connection cap on active commands. The default is `128`. This works with the global cap to prevent one multiplexed client from monopolizing the server while allowing other connections to make progress.

### `limits.max_inflight_payload_bytes`

Aggregate budget for inbound payload bytes associated with active requests. The default is `268435456` bytes (256 MiB). It must be at least `max_payload_bytes`; otherwise a single legal frame could never be admitted. Lower it when many concurrent clients can submit large requests.

### `limits.max_pending_upload_bytes`

Maximum total declared size reserved by active chunked blob uploads. The default is `1073741824` bytes (1 GiB). This protects disk spool and memory-accounting paths from many unfinished uploads. It is separate from the maximum size of one frame and the AOF path.

### `limits.upload_ttl_ms`

Inactivity lifetime for an unfinished blob upload. The default is `600000` milliseconds (10 minutes), and the minimum is `1000` milliseconds. Expired uploads are discarded; lower the value when clients are unreliable, or raise it for genuinely slow upload links while keeping the aggregate pending-upload budget bounded.

### `limits.command_timeout_ms`

Server-side deadline for one command. The default is `30000` milliseconds and the value must be positive. It also bounds blocking data-type operations at the protocol request level. This is separate from the client's own dial/request timeout; both deadlines can terminate a request.

## In-memory compression

Compression is applied to supported contiguous values after they cross the configured size threshold. The compressed representation is retained only when it is smaller enough to satisfy the savings threshold; enabling compression therefore does not force every value into a compressed form.

### `compression.enabled`

Master switch for adaptive in-memory compression. The default is `true`. Disable it when CPU latency matters more than memory, or when the workload consists mostly of values that do not compress well.

### `compression.algorithm`

Controls the compressor selection. `auto` tries the low-latency RLE and LZ4 candidates and keeps the smaller result; `lz4` favors fast block compression; `rle` favors repeated-byte data; `deflate-fast` uses a fast DEFLATE setting; `none` disables compression regardless of the enabled flag. Values are validated at startup.

### `compression.min_bytes`

Minimum original value size eligible for compression. The default is `1024` bytes. Smaller values remain in their native representation to avoid metadata and CPU overhead. The value must be non-negative.

### `compression.min_savings_percent`

Minimum percentage reduction required before a compressed representation is kept. The default is `8`; valid values are at least `0` and below `100`. A high threshold reduces CPU and representation churn for marginal wins; a low threshold saves more memory at the cost of compressing data that barely shrinks.

## Filesystem root

This directory contains runtime state that is separate from the primary JSON configuration and the AOF journal.

### `data_dir`

Root directory for runtime files that are not the primary JSON configuration: generated bootstrap-token material, `admin.json` control-plane state, upload spools, checkpoint/backups, and other runtime data. Amaquet creates it with restrictive directory permissions during startup. Put it on persistent storage when administration state or upload recovery must survive process replacement, and back it up separately from the AOF journal.

## Choosing a safe starting profile

For a single-process local development server, keep both hosts on `127.0.0.1`, use the default authentication and TLS settings, and use `amaquet --init-config` so the bootstrap secret is generated rather than invented. For a networked deployment, use private listener addresses, TLS on both surfaces, strong API keys, an explicit persistent `data_dir`, AOF settings appropriate to the recovery objective, and memory/connection limits sized below the host or container ceiling.

## Environment overrides

The complete environment-override surface is below. Numeric values that cannot be parsed are ignored. For normal boolean overrides, only the exact strings `1` and `true` enable the option. Compression is the inverse form: it is enabled unless the value is exactly `0` or `false`.

| Variable                              | Configuration path / behavior                       |
| ------------------------------------- | --------------------------------------------------- |
| `AMAQUET_HOST`                        | `protocol.host`                                     |
| `AMAQUET_PORT`                        | `protocol.port`                                     |
| `AMAQUET_ADMIN_HOST`                  | `admin.host`                                        |
| `AMAQUET_ADMIN_PORT`                  | `admin.port`                                        |
| `AMAQUET_ALLOW_INSECURE_AUTH`         | `protocol.allow_insecure_auth`                      |
| `AMAQUET_ADMIN_ALLOW_INSECURE`        | `admin.allow_insecure`                              |
| `AMAQUET_BOOTSTRAP_TOKEN`             | `security.bootstrap_token` at runtime               |
| `AMAQUET_TLS_CERT`                    | `protocol.tls_cert_file`; also enables protocol TLS |
| `AMAQUET_TLS_KEY`                     | `protocol.tls_key_file`; also enables protocol TLS  |
| `AMAQUET_COMPRESSION`                 | `compression.enabled`                               |
| `AMAQUET_COMPRESSION_ALGORITHM`       | `compression.algorithm`                             |
| `AMAQUET_COMPRESSION_MIN_BYTES`       | `compression.min_bytes`                             |
| `AMAQUET_AOF_ENABLED`                 | `persistence.aof_enabled`                           |
| `AMAQUET_AOF_PATH`                    | `persistence.aof_path`                              |
| `AMAQUET_AOF_FSYNC`                   | `persistence.fsync`                                 |
| `AMAQUET_MAX_MEMORY_BYTES`            | `limits.max_memory_bytes`                           |
| `AMAQUET_EVICTION_POLICY`             | `limits.eviction_policy`                            |
| `AMAQUET_MAX_CONNECTIONS`             | `limits.max_connections`                            |
| `AMAQUET_MAX_INFLIGHT_REQUESTS`       | `limits.max_inflight_requests`                      |
| `AMAQUET_MAX_INFLIGHT_PER_CONNECTION` | `limits.max_inflight_per_connection`                |
| `AMAQUET_MAX_INFLIGHT_PAYLOAD_BYTES`  | `limits.max_inflight_payload_bytes`                 |
| `AMAQUET_MAX_PENDING_UPLOAD_BYTES`    | `limits.max_pending_upload_bytes`                   |
| `AMAQUET_UPLOAD_TTL_MS`               | `limits.upload_ttl_ms`                              |
| `AMAQUET_COMMAND_TIMEOUT_MS`          | `limits.command_timeout_ms`                         |

There are no environment overrides for admin TLS files, `max_payload_bytes`, `max_sessions`, `auth_failures_per_minute`, or `compression.min_savings_percent`; configure those in JSON.

An environment-provided bootstrap secret is runtime-only. An API configuration save does not copy that secret into the JSON configuration file.

## Restart behavior

The control-plane system endpoint compares the full public configuration. It persists rather than hot-applies changes, so any submitted public configuration that differs from the running configuration is reported with `restart_required: true`.

Validation rejects insecure non-loopback authenticated protocol and admin listeners unless the matching explicit insecure override is enabled.

## Validation constraints

Both listener ports must be between 1 and 65535. Connection, in-flight request, pending-upload, command-timeout, HTTP-session, and authentication-rate limits must be positive. `max_payload_bytes` is limited to 1 byte through 64 MiB; the aggregate in-flight payload budget must be at least that large. `max_memory_bytes` and protocol timeouts may be zero but not negative, and upload TTL must be at least 1000 ms.

TLS requires both certificate and key paths. Compression minimum bytes cannot be negative, and its required saving must be at least zero and below 100 percent. Enumerated persistence, compression, and eviction values must match the options in the table above.
