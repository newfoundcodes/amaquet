---
title: "inverted_index"
---

**Category:** Indexes and search  
**Wire type:** `inverted_index`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:inverted_index","type":"inverted_index","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`INDEX`, `REMOVE`, `SEARCH`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:inverted_index` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:inverted_index",
  "type": "inverted_index",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP INDEX`

Accepts `id`, string `text` and returns `true`; replaces previous terms for ID.

```json
{
  "key": "example:inverted_index",
  "operation": "INDEX",
  "args": {
    "id": "d1",
    "text": "amaquet in-memory database"
  }
}
```

#### `OP SEARCH`

Accepts string `query`, optional `limit:20` and returns hits sorted by descending TF×IDF-like score, then ID.

```json
{
  "key": "example:inverted_index",
  "operation": "SEARCH",
  "args": {
    "query": "memory",
    "limit": 10
  }
}
```

#### `OP REMOVE`

Accepts `id` and returns `true` even when the document is absent.

```json
{
  "key": "example:inverted_index",
  "operation": "REMOVE",
  "args": {
    "id": "d1"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:inverted_index"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:inverted_index"}` — returns the wire type.
- `EXISTS`: `{"key":"example:inverted_index"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:inverted_index","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:inverted_index"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:inverted_index"}` — removes the TTL.
- `DEL`: `{"keys":["example:inverted_index"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
