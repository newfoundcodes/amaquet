---
title: "rate_limit_bucket"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `rate_limit_bucket`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:rate_limit_bucket","type":"rate_limit_bucket","options":{"capacity":5,"refill_per_second":2}}'
```

## Construction options

Example `CREATE` options: `{"capacity":5,"refill_per_second":2}`. Omitted fields use server defaults.

## Supported operations

`ALLOW`, `REMAINING`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:rate_limit_bucket` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:rate_limit_bucket",
  "type": "rate_limit_bucket",
  "options": {
    "capacity": 5,
    "refill_per_second": 2
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ALLOW`

Accepts optional numeric `cost:1` and returns boolean; non-positive cost is allowed without consuming tokens.

```json
{
  "key": "example:rate_limit_bucket",
  "operation": "ALLOW",
  "args": {
    "cost": 1
  }
}
```

#### `OP REMAINING`

Takes no arguments and returns current tokens after time-based refill.

```json
{
  "key": "example:rate_limit_bucket",
  "operation": "REMAINING",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:rate_limit_bucket"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:rate_limit_bucket"}` — returns the wire type.
- `EXISTS`: `{"key":"example:rate_limit_bucket"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:rate_limit_bucket","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:rate_limit_bucket"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:rate_limit_bucket"}` — removes the TTL.
- `DEL`: `{"keys":["example:rate_limit_bucket"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
