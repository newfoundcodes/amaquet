---
title: "Installation"
description: "Install Amaquet from source, configure its server and CLI binaries, secure the first startup, and verify an authenticated request."
---

This guide builds Amaquet from the repository, creates a secure local configuration, starts the server, and verifies the first authenticated request. Amaquet is a source-built Go server with a separate command-line client, key generator, and offline restore utility; it does not require Redis or another database.

## What gets installed

The normal build produces four executables in `bin/`:

| Executable            | Purpose                                                                                  |
| --------------------- | ---------------------------------------------------------------------------------------- |
| `bin/amaquet`         | Database server, native protocol listener, and HTTP administration listener.             |
| `bin/amaquet-cli`     | Command-line client for native protocol commands and type operations.                    |
| `bin/amaquet-keygen`  | Generates the optional Ed25519 server identity key pair.                                 |
| `bin/amaquet-restore` | Validates and atomically installs a verified AOF checkpoint while the server is stopped. |

The server keeps active values in memory. AOF persistence and the control-plane files under `data_dir` are optional but should be placed on persistent, protected storage for a durable deployment.

## Prerequisites

Source builds require Go 1.23 or newer. Git is needed to clone the repository. Node.js 22.12 or newer and npm are only required when building or serving the Astro documentation site; they are not required to run the Amaquet server or CLI.

Check the installed toolchains before starting:

```bash
go version
git --version
node --version   # only required for docs
npm --version    # only required for docs
```

The server needs permission to bind its configured protocol and admin ports, create `data_dir`, and read any configured TLS or identity key files. A production service also needs enough file descriptors for its connection limit and enough memory for the engine plus runtime overhead.

## Obtain the source

Clone the repository and work from its root so the default relative paths such as `./amaquet.json` and `./data` resolve predictably:

```bash
git clone https://github.com/newfoundcodes/amaquet.git
cd amaquet
```

If the source is already checked out, update it and review the release or deployment changes before rebuilding. Keep the existing `data_dir` and AOF path when upgrading an installation that must retain its data.

## Build the binaries

The recommended build creates `bin/` and writes all four executables there:

```bash
make build
```

The equivalent explicit commands are:

```bash
mkdir -p bin
go build -trimpath -o bin/amaquet ./cmd/amaquet
go build -trimpath -o bin/amaquet-cli ./cmd/amaquet-cli
go build -trimpath -o bin/amaquet-keygen ./cmd/amaquet-keygen
go build -trimpath -o bin/amaquet-restore ./cmd/amaquet-restore
```

Confirm that the binaries are executable and report the expected server version:

```bash
./bin/amaquet -version
./bin/amaquet-cli -h
```

`make build` does not install files into a system directory. Either invoke the binaries with their `./bin/` paths or add the repository's `bin/` directory to a controlled service or shell `PATH`.

## Create the first configuration

Generate a default configuration and a high-entropy bootstrap administrator credential:

```bash
./bin/amaquet --init-config --config ./amaquet.json
```

The command writes the JSON file, prints the bootstrap token once, and exits. Save the token in a password manager or secret store immediately. The generated file contains the token and is written with restrictive permissions; do not commit it or include it in a public support bundle.

The generated defaults bind both listeners to loopback, require native protocol authentication, keep AOF disabled, and use `./data` for runtime state. Review the file before starting the server. The complete field-by-field explanation is in [Configuration](configuration.md); for a first local installation, the generated defaults are safer than copying a network-exposed example.

If `security.bootstrap_token` is empty during normal startup, Amaquet reuses `data_dir/bootstrap-token` or creates it with mode `0600`. Startup logs the protected path rather than printing the new secret. This fallback is useful for unattended first boot, but a managed deployment should inject or provision a deliberate secret and rotate it into a permanent API key.

## Start a local server

Start the server in the foreground so startup errors and the listener addresses remain visible:

```bash
./bin/amaquet --config ./amaquet.json
```

With the generated defaults, the endpoints are:

```text
Native protocol: amaquet://127.0.0.1:13378
Admin API:       http://127.0.0.1:13379
Data directory:  ./data
```

The process creates `data_dir`, loads the administration state, optionally replays the AOF, and then starts both listeners. Keep the process running while verifying it. Press `Ctrl-C` for a graceful shutdown; the server closes listeners and flushes the AOF according to its configured durability mode.

## Verify the installation

The admin health endpoint is a convenient first check because it confirms that the HTTP listener is responding:

```bash
curl --fail --silent --show-error http://127.0.0.1:13379/api/health
```

Use the bootstrap token printed during initialization for the first native protocol request. Store it in a shell variable without placing it directly into a committed script:

```bash
export AMAQUET_BOOTSTRAP_TOKEN='paste-the-token-from-initialization'
./bin/amaquet-cli \
  -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  PING '{}'
```

A successful response confirms network reachability, protocol negotiation, and authentication. Use the same credential to inspect the server and type registry:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" INFO '{}'
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" TYPES '{}'
```

For a complete data-plane check, store and read one typed value:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  SET '{"key":"install:check","type":"utf8_string","value":"Amaquet is running"}'

./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 -api-key "$AMAQUET_BOOTSTRAP_TOKEN" \
  GET '{"key":"install:check"}'
```

## Create a permanent API key

The bootstrap credential is intended for initial control-plane setup, not for application traffic. Use the authenticated admin API to create a permanent API key with the narrowest role and permissions required by each application. Amaquet stores hashes of generated API-key secrets and shows each raw secret only once. After the first admin API key is created, bootstrap authentication is permanently disabled in the persisted administration state.

Follow [Admin API](../reference/admin-api.md) for the exact API-key request and role model. Keep the generated secret outside `amaquet.json`, provide it to clients through a secret manager, and use `-api-key` or the Go client's API-key option rather than embedding credentials in a URI.

## Configure network access and TLS

The generated configuration is intentionally loopback-only. Before exposing either listener, read [Configuration](configuration.md) and [Production deployment](../operations/deployment.md):

- Use a specific private bind address instead of `0.0.0.0` when the topology allows it.
- Enable `protocol.tls` for native traffic and use `amaquets://` client URIs.
- Enable `admin.tls` for the administration API and protect its private key.
- Keep `protocol.require_auth` and `admin.metrics_require_auth` enabled.
- Leave `protocol.allow_insecure_auth` and `admin.allow_insecure` disabled unless a deliberately isolated network requires them.
- Restrict firewall access to application and management networks, and monitor `/api/ready` and `/metrics` according to the authentication policy.

Generate the optional Ed25519 protocol identity separately from TLS certificates:

```bash
mkdir -p keys
./bin/amaquet-keygen \
  --public ./keys/amaquet-public.pem \
  --private ./keys/amaquet-private.pem
```

Set both generated paths under `security.public_key_file` and `security.private_key_file` when clients need application-level identity verification. The key pair does not replace an X.509 certificate, does not encrypt traffic, and should be readable only by the service account where possible.

## Environment-based deployment

JSON is convenient for a checked-in template, while environment overrides are useful for containers and service managers. Supported variables are documented in [Configuration](configuration.md). Overrides are applied at startup after JSON is loaded; invalid numeric values are ignored, while the final merged configuration is still validated.

For example, this starts with a JSON file but moves the secret and listener choices into the process environment:

```bash
export AMAQUET_BOOTSTRAP_TOKEN="$BOOTSTRAP_SECRET"
export AMAQUET_HOST=127.0.0.1
export AMAQUET_PORT=13378
export AMAQUET_ADMIN_HOST=127.0.0.1
export AMAQUET_ADMIN_PORT=13379
./bin/amaquet --config /etc/amaquet/amaquet.json
```

Do not assume that an environment override is persisted. A later restart needs the same environment, a JSON value, or the generated `data_dir/bootstrap-token` fallback.

## Docker installation

The repository includes development and production-oriented Compose files. Build and start the loopback-only development deployment with:

```bash
docker compose build
docker compose up -d
docker compose logs -f amaquet
```

The Compose configuration persists `/app/data` in the `amaquet-data` volume. Do not remove that volume if AOF data or administrator state must survive container replacement. The production Compose file is a template: provide the expected TLS and identity files, review host bindings and firewall rules, and inject the bootstrap secret through the deployment platform. See [Docker deployment](../operations/docker.md) for the port, volume, user, and TLS details.

## Documentation tools

Node.js and npm are needed only for the documentation site. From the repository root:

```bash
make docs-install
make docs-build
make docs-serve
```

`make docs-build` validates the source-derived documentation inventories, checks heading descriptions, builds the static site, and checks generated links. `make docs-serve` starts the local Astro development server for editing the documentation; it does not start Amaquet.

## Upgrades and backups

Before replacing a server binary or changing persistence settings, stop Amaquet cleanly and back up both the AOF and `data_dir`, including `admin.json`, bootstrap-token material, upload state, and identity files. Keep the configuration path and AOF path stable unless the move is deliberate.

For an AOF checkpoint, use the administration persistence endpoint and verify the resulting file before storing it separately. To install a verified checkpoint, stop the target process and use `amaquet-restore`; never copy over a live AOF while the server is writing it. Read [Persistence and recovery](../concepts/persistence.md) and [Backup and recovery](../operations/backups.md) before treating AOF as a backup strategy.

## Troubleshooting first startup

Use this table to narrow down failures during the first launch. Once the process is running, the server log and the health endpoint usually provide the next useful detail.

| Symptom                                                        | Likely cause and action                                                                                                                                                           |
| -------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Failed to load configuration`                                 | The JSON is malformed or the `-config` path is wrong. Validate the file and use an absolute path while diagnosing.                                                                |
| `invalid port`                                                 | A listener port is outside `1`–`65535` or collides with another process.                                                                                                          |
| `TLS requires ...`                                             | TLS is enabled without both certificate and key paths, or the process cannot read one of the files.                                                                               |
| `authenticated Amaquet on a non-loopback address requires TLS` | Enable native TLS or deliberately set `protocol.allow_insecure_auth`; TLS is the recommended fix.                                                                                 |
| `admin API on a non-loopback address requires TLS`             | Enable admin HTTPS or keep the admin host on loopback/private management access.                                                                                                  |
| CLI connection refused                                         | Confirm the server is running, the URI port matches `protocol.port`, and the listener is bound to an address reachable from the client.                                           |
| authentication failure                                         | Use the token printed by `--init-config`, the current `data_dir/bootstrap-token`, or a permanent API key; do not assume a JSON token is still active after bootstrap is disabled. |
| AOF replay or permission failure                               | Stop the process, verify ownership and permissions of the AOF/data directory, preserve the original files, and follow the recovery guide before retrying.                         |

## Next steps

Continue with [Quick start](quickstart.md) for typed values and operations, [Configuration](configuration.md) for every server setting, [Authentication and identity](../protocol/authentication.md) for protocol credentials, and [Production deployment](../operations/deployment.md) for supervision and durable service layouts.
