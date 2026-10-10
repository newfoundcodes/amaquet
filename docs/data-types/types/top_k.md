---
title: "top_k"
---

**Category:** Probabilistic  
**Wire type:** `top_k`  
**Construction:** `CREATE`

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:top_k","type":"top_k","options":{"k":5}}'
```

## Construction options

Example `CREATE` options: `{"k":5}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `ITEMS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:top_k` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:top_k",
  "type": "top_k",
  "options": {
    "k": 5
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts string `value`, optional unsigned `count:1` and returns count-Min estimate.

```json
{
  "key": "example:top_k",
  "operation": "ADD",
  "args": {
    "value": "alice",
    "count": 10
  }
}
```

#### `OP ITEMS`

Takes no arguments and returns up to `k` candidates by estimated count, then lexical tie-break.

```json
{
  "key": "example:top_k",
  "operation": "ITEMS",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:top_k"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:top_k"}` — returns the wire type.
- `EXISTS`: `{"key":"example:top_k"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:top_k","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:top_k"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:top_k"}` — removes the TTL.
- `DEL`: `{"keys":["example:top_k"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
