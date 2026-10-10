---
title: "lock"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `lock`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:lock","type":"lock","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`TRY_LOCK`, `RENEW`, `UNLOCK`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:lock` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:lock",
  "type": "lock",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP TRY_LOCK`

Accepts `owner`, `token`, optional `ttl_ms:30000` and returns boolean; implemented by a lease.

```json
{
  "key": "example:lock",
  "operation": "TRY_LOCK",
  "args": {
    "owner": "worker-a",
    "token": "t1",
    "ttl_ms": 5000
  }
}
```

#### `OP UNLOCK`

Accepts `token`; renew accepts `ttl_ms:30000` and returns boolean.

```json
{
  "key": "example:lock",
  "operation": "UNLOCK",
  "args": {
    "token": "t1"
  }
}
```

#### `OP RENEW`

Accepts `token` and optional `ttl_ms`; returns the token-checked boolean result.

```json
{
  "key": "example:lock",
  "operation": "RENEW",
  "args": {
    "token": "t1",
    "ttl_ms": 5000
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:lock"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:lock"}` — returns the wire type.
- `EXISTS`: `{"key":"example:lock"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:lock","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:lock"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:lock"}` — removes the TTL.
- `DEL`: `{"keys":["example:lock"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
