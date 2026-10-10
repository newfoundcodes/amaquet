---
title: "hyperloglog"
---

**Category:** Probabilistic  
**Wire type:** `hyperloglog`  
**Construction:** `CREATE`

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:hyperloglog","type":"hyperloglog","options":{"precision":12}}'
```

## Construction options

Example `CREATE` options: `{"precision":12}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `COUNT`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:hyperloglog` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:hyperloglog",
  "type": "hyperloglog",
  "options": {
    "precision": 12
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts string `value` and returns whether a register increased.

```json
{
  "key": "example:hyperloglog",
  "operation": "ADD",
  "args": {
    "value": "alice"
  }
}
```

#### `OP COUNT`

Takes no arguments and returns rounded approximate cardinality with small-range linear counting.

```json
{
  "key": "example:hyperloglog",
  "operation": "COUNT",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:hyperloglog"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:hyperloglog"}` — returns the wire type.
- `EXISTS`: `{"key":"example:hyperloglog"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:hyperloglog","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:hyperloglog"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:hyperloglog"}` — removes the TTL.
- `DEL`: `{"keys":["example:hyperloglog"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
