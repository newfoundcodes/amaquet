---
title: "count_min_sketch"
---

**Category:** Probabilistic  
**Wire type:** `count_min_sketch`  
**Construction:** `CREATE`

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:count_min_sketch","type":"count_min_sketch","options":{"width":256,"depth":4}}'
```

## Construction options

Example `CREATE` options: `{"width":256,"depth":4}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `ESTIMATE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:count_min_sketch` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:count_min_sketch",
  "type": "count_min_sketch",
  "options": {
    "width": 256,
    "depth": 4
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts string `value`, optional unsigned `count:1` and returns current minimum counter estimate.

```json
{
  "key": "example:count_min_sketch",
  "operation": "ADD",
  "args": {
    "value": "alice",
    "count": 5
  }
}
```

#### `OP ESTIMATE`

Accepts string `value` and returns approximate frequency (never below inserted count absent overflow).

```json
{
  "key": "example:count_min_sketch",
  "operation": "ESTIMATE",
  "args": {
    "value": "alice"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:count_min_sketch"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:count_min_sketch"}` — returns the wire type.
- `EXISTS`: `{"key":"example:count_min_sketch"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:count_min_sketch","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:count_min_sketch"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:count_min_sketch"}` — removes the TTL.
- `DEL`: `{"keys":["example:count_min_sketch"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
