---
title: "vector_set"
---

**Category:** Vectors  
**Wire type:** `vector_set`  
**Construction:** `CREATE`

Vector types support embeddings, metadata, quantization, and similarity search.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:vector_set","type":"vector_set","options":{"dim":3,"metric":"cosine"}}'
```

## Construction options

`dim` is required and must be positive. `metric` defaults to `cosine`; portable values are `cosine`, `dot`, `euclidean`, and `manhattan`, with unsupported strings rejected when distance calculation is first required.

## Supported operations

`ADD`, `REMOVE`, `SEARCH`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:vector_set` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:vector_set",
  "type": "vector_set",
  "options": {
    "dim": 3,
    "metric": "cosine"
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `id`, `vector`, optional `metadata` and returns `true`; vector dimension enforced; replaces same ID.

```json
{
  "key": "example:vector_set",
  "operation": "ADD",
  "args": {
    "id": "v1",
    "vector": [
      1,
      0,
      0
    ],
    "metadata": {
      "class": "a"
    }
  }
}
```

#### `OP SEARCH`

Accepts `vector`, optional `k:10` and returns exact full-scan results sorted by ascending distance.

```json
{
  "key": "example:vector_set",
  "operation": "SEARCH",
  "args": {
    "vector": [
      1,
      0,
      0
    ],
    "k": 1
  }
}
```

#### `OP REMOVE`

Accepts `id` and returns whether the vector existed.

```json
{
  "key": "example:vector_set",
  "operation": "REMOVE",
  "args": {
    "id": "v1"
  }
}
```

#### `OP LEN`

Takes no arguments and returns the item count.

```json
{
  "key": "example:vector_set",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:vector_set"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:vector_set"}` — returns the wire type.
- `EXISTS`: `{"key":"example:vector_set"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:vector_set","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:vector_set"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:vector_set"}` — removes the TTL.
- `DEL`: `{"keys":["example:vector_set"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
