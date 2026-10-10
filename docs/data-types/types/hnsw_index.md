---
title: "hnsw_index"
---

**Category:** Indexes and search  
**Wire type:** `hnsw_index`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:hnsw_index","type":"hnsw_index","options":{"dim":3,"metric":"cosine","m":8,"ef_search":32}}'
```

## Construction options

`dim` is required and must be positive. `metric` defaults to `cosine`; portable values are `cosine`, `dot`, `euclidean`, and `manhattan`, with unsupported strings rejected when distance calculation is first required. `m` defaults to 16, and `ef_search` defaults to 64.

## Supported operations

`ADD`, `REMOVE`, `SEARCH`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:hnsw_index` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:hnsw_index",
  "type": "hnsw_index",
  "options": {
    "dim": 3,
    "metric": "cosine",
    "m": 8,
    "ef_search": 32
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `id`, `vector`, optional `metadata` and returns `true`; replaces same ID and reconnects graph.

```json
{
  "key": "example:hnsw_index",
  "operation": "ADD",
  "args": {
    "id": "v1",
    "vector": [
      1,
      0,
      0
    ],
    "metadata": {}
  }
}
```

#### `OP SEARCH`

Accepts `vector`, optional `k:10` and returns approximate ascending-distance neighbors.

```json
{
  "key": "example:hnsw_index",
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
  "key": "example:hnsw_index",
  "operation": "REMOVE",
  "args": {
    "id": "v1"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:hnsw_index"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:hnsw_index"}` — returns the wire type.
- `EXISTS`: `{"key":"example:hnsw_index"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:hnsw_index","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:hnsw_index"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:hnsw_index"}` — removes the TTL.
- `DEL`: `{"keys":["example:hnsw_index"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
