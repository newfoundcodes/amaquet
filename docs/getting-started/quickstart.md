---
title: "Quick start"
description: "Run Amaquet locally, create a first key, connect with amaquet-cli, and move from an unauthenticated example to a secure deployment."
---

This quick start keeps both listeners on loopback, but leaves native authentication enabled. It uses the same bootstrap flow as a production installation: generate a bootstrap credential, create a permanent API key, then use that key for application traffic.

## 1. Build and initialize

Build the binaries and generate a complete local configuration:

```bash
make build
./bin/amaquet --init-config --config ./amaquet.json
```

`--init-config` writes a random bootstrap token into the configuration, prints it once, and exits. Store it in a protected shell environment for the rest of this guide; do not commit the configuration file.

```bash
export AMAQUET_BOOTSTRAP_TOKEN='paste-the-token-printed-by-init-config'
```

The generated local configuration keeps both listeners on `127.0.0.1`, requires native authentication, and leaves AOF disabled by default. Enable `persistence.aof_enabled` and keep the default `everysec` fsync mode when restart recovery matters.

## 2. Start Amaquet

Start the server with the local configuration created in the previous step. The process remains in the foreground and owns both configured listeners.

```bash
./bin/amaquet --config ./amaquet.json
```

## 3. Create a permanent API key

The bootstrap token is intended for initialization, not routine application access. Use the admin API to create an `admin` API key. The secret is returned only in this response.

```bash
curl --fail --silent \
  -H "Authorization: Bearer $AMAQUET_BOOTSTRAP_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"local-admin","role":"admin"}' \
  http://127.0.0.1:13379/api/api-keys
```

Copy the `secret` field from the response and store it outside the repository:

```bash
export AMAQUET_API_KEY='paste-the-secret-from-the-response'
```

Creating the first admin API key permanently disables bootstrap authentication in the persisted administration state. If you lose every administrator credential, recovery requires restoring administration state from backup or performing an intentional state-recovery procedure.

## 4. Ping

From another terminal, use the CLI to verify that the native protocol listener accepts commands.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" PING '{}'
```

## 5. Store a typed scalar

Store a directly encoded value with `SET`, then retrieve it with `GET` to confirm its type and payload.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" SET \
  '{"key":"user:1:name","type":"utf8_string","value":"Nathanne"}'
```

Read it:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" GET \
  '{"key":"user:1:name"}'
```

## 6. Create a collection

Composite values are initialized with `CREATE`; use `OP` to apply a type-specific mutation afterward.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" CREATE \
  '{"key":"queue:jobs","type":"fifo_queue","options":{}}'
```

Add a typed value:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" OP \
  '{"key":"queue:jobs","operation":"ENQUEUE","args":{"value":{"type":"utf8_string","value":"build"}}}'
```

## 7. Add TTL

Expiration is expressed in positive milliseconds. This command gives the scalar key a 60-second lifetime without changing its value.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" EXPIRE \
  '{"key":"user:1:name","ttl_ms":60000}'
```

## 8. Inspect server metadata

Use these read-only commands to inspect runtime information and the registered type catalog.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" INFO '{}'
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_API_KEY" TYPES '{}'
```

Next, read [Amaquet commands](../protocol/commands.md), [data types](../data-types/index.md), and [configuration](configuration.md). For a disposable unauthenticated test process, set `protocol.require_auth` to `false` explicitly and keep both listeners on loopback.
