---
title: "sparse_vector"
---

**Category:** Vectors  
**Wire type:** `sparse_vector`  
**Construction:** `SET`

Vector types support embeddings, metadata, quantization, and similarity search.

## Create a value

Use the top-level `SET` command to store this directly encoded value. The example shows the required wire representation.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:sparse_vector","type":"sparse_vector","value":{"dim":8,"values":{"1":0.5,"5":1.25}}}'
```

## Construction options

This type is constructed with `SET`; it has no `CREATE` options.

## Supported operations

`GET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:sparse_vector` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `SET`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:sparse_vector",
  "type": "sparse_vector",
  "value": {
    "dim": 8,
    "values": {
      "1": 0.5,
      "5": 1.25
    }
  }
}
```

### Operations

The response is the type-specific value described in the [complete operation contract](../operations.md).

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:sparse_vector"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:sparse_vector"}` — returns the wire type.
- `EXISTS`: `{"key":"example:sparse_vector"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:sparse_vector","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:sparse_vector"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:sparse_vector"}` — removes the TTL.
- `DEL`: `{"keys":["example:sparse_vector"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
