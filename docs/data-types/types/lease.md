---
title: "lease"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `lease`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:lease","type":"lease","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ACQUIRE`, `RENEW`, `RELEASE`, `HELD`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:lease` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:lease",
  "type": "lease",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ACQUIRE`

Accepts `owner`, `token`, optional `ttl_ms:30000` and returns boolean; only when absent/expired and TTL positive.

```json
{
  "key": "example:lease",
  "operation": "ACQUIRE",
  "args": {
    "owner": "worker-a",
    "token": "t1",
    "ttl_ms": 5000
  }
}
```

#### `OP HELD`

Takes no arguments and returns whether a non-expired token exists.

```json
{
  "key": "example:lease",
  "operation": "HELD",
  "args": {}
}
```

#### `OP RELEASE`

Accepts `token`; renew accepts `ttl_ms:30000` and returns boolean token-checked result.

```json
{
  "key": "example:lease",
  "operation": "RELEASE",
  "args": {
    "token": "t1"
  }
}
```

#### `OP RENEW`

Accepts `token` and optional `ttl_ms`; returns the token-checked boolean result.

```json
{
  "key": "example:lease",
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
  "key": "example:lease"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:lease"}` — returns the wire type.
- `EXISTS`: `{"key":"example:lease"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:lease","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:lease"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:lease"}` — removes the TTL.
- `DEL`: `{"keys":["example:lease"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
