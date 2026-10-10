---
title: "t_digest"
---

**Category:** Probabilistic  
**Wire type:** `t_digest`  
**Construction:** `CREATE`

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:t_digest","type":"t_digest","options":{"compression":100}}'
```

## Construction options

Example `CREATE` options: `{"compression":100}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `QUANTILE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:t_digest` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:t_digest",
  "type": "t_digest",
  "options": {
    "compression": 100
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts numeric `value`, optional `weight:1` and returns `true`; non-positive weights are ignored internally.

```json
{
  "key": "example:t_digest",
  "operation": "ADD",
  "args": {
    "value": 5
  }
}
```

#### `OP QUANTILE`

Accepts numeric `q` and returns approximate quantile; q clamps to 0…1; empty digest returns JSON-incompatible NaN and therefore should be avoided.

```json
{
  "key": "example:t_digest",
  "operation": "QUANTILE",
  "args": {
    "q": 0.5
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:t_digest"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:t_digest"}` — returns the wire type.
- `EXISTS`: `{"key":"example:t_digest"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:t_digest","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:t_digest"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:t_digest"}` — removes the TTL.
- `DEL`: `{"keys":["example:t_digest"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
