---
title: "Quick start"
description: "Run Amaquet locally, create a first key, connect with amaquet-cli, and move from an unauthenticated example to a secure deployment."
---

This example uses authentication disabled only to keep the first local session short. Production deployments should enable authentication and TLS.

## 1. Local configuration

Create `amaquet.json`:

```json
{
  "protocol": {
    "host": "127.0.0.1",
    "port": 13378,
    "tls": false,
    "require_auth": false
  },
  "admin": {
    "host": "127.0.0.1",
    "port": 13379
  },
  "security": {
    "bootstrap_token": "replace-this"
  },
  "persistence": {
    "aof_enabled": false,
    "aof_path": "./data/amaquet.aof",
    "fsync": "everysec"
  },
  "limits": {
    "max_connections": 10000,
    "max_payload_bytes": 67108864
  },
  "data_dir": "./data"
}
```

## 2. Start Amaquet

Start the server with the local configuration created in the previous step. The process remains in the foreground and owns both configured listeners.

```bash
./bin/amaquet -config ./amaquet.json
```

## 3. Ping

From another terminal, use the CLI to verify that the native protocol listener accepts commands.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 PING '{}'
```

## 4. Store a typed scalar

Store a directly encoded value with `SET`, then retrieve it with `GET` to confirm its type and payload.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 SET \
  '{"key":"user:1:name","type":"utf8_string","value":"Nathanne"}'
```

Read it:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 GET \
  '{"key":"user:1:name"}'
```

## 5. Create a collection

Composite values are initialized with `CREATE`; use `OP` to apply a type-specific mutation afterward.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE \
  '{"key":"queue:jobs","type":"fifo_queue","options":{}}'
```

Add a typed value:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 OP \
  '{"key":"queue:jobs","operation":"ENQUEUE","args":{"value":{"type":"utf8_string","value":"build"}}}'
```

## 6. Add TTL

Expiration is expressed in positive milliseconds. This command gives the scalar key a 60-second lifetime without changing its value.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 EXPIRE \
  '{"key":"user:1:name","ttl_ms":60000}'
```

## 7. Inspect server metadata

Use these read-only commands to inspect runtime information and the registered type catalog.

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 INFO '{}'
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 TYPES '{}'
```

Next, read [Amaquet commands](../protocol/commands.md) and [data types](../data-types/index.md).
