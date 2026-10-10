---
title: "dense_vector"
---

**Category:** Vectors  
**Wire type:** `dense_vector`  
**Construction:** `SET`

Vector types support embeddings, metadata, quantization, and similarity search.

## Create a value

Use the top-level `SET` command to store this directly encoded value. The example shows the required wire representation.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:dense_vector","type":"dense_vector","value":[1.0,0.0,0.5]}'
```

## Construction options

This type is constructed with `SET`; it has no `CREATE` options.

## Supported operations

`DISTANCE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:dense_vector` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `SET`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:dense_vector",
  "type": "dense_vector",
  "value": [
    1.0,
    0.0,
    0.5
  ]
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP DISTANCE`

Accepts `vector:[...]`, optional `metric:"cosine"` and returns numeric distance; dimensions must match.

```json
{
  "key": "example:dense_vector",
  "operation": "DISTANCE",
  "args": {
    "vector": [
      1.0,
      0.0,
      0.5
    ],
    "metric": "cosine"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:dense_vector"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:dense_vector"}` — returns the wire type.
- `EXISTS`: `{"key":"example:dense_vector"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:dense_vector","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:dense_vector"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:dense_vector"}` — removes the TTL.
- `DEL`: `{"keys":["example:dense_vector"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
