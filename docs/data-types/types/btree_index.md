---
title: "btree_index"
---

**Category:** Indexes and search  
**Wire type:** `btree_index`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:btree_index","type":"btree_index","options":{"degree":8}}'
```

## Construction options

Example `CREATE` options: `{"degree":8}`. Omitted fields use server defaults.

## Supported operations

`INSERT`, `DELETE`, `SEARCH`, `RANGE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:btree_index` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:btree_index",
  "type": "btree_index",
  "options": {
    "degree": 8
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP INSERT`

Accepts string `value`, `id` and returns `true`.

```json
{
  "key": "example:btree_index",
  "operation": "INSERT",
  "args": {
    "value": "b",
    "id": "2"
  }
}
```

#### `OP SEARCH`

Accepts string `value` and returns lexically sorted IDs for exact key.

```json
{
  "key": "example:btree_index",
  "operation": "SEARCH",
  "args": {
    "value": "a"
  }
}
```

#### `OP DELETE`

Accepts string `value` and `id` and returns whether that association existed and was removed.

```json
{
  "key": "example:btree_index",
  "operation": "DELETE",
  "args": {
    "value": "b",
    "id": "2"
  }
}
```

#### `OP RANGE`

Accepts an inclusive minimum `value` and optional `max` and `limit`; returns a map of matching keys to sorted IDs.

```json
{
  "key": "example:btree_index",
  "operation": "RANGE",
  "args": {
    "value": "a",
    "max": "z",
    "limit": 10
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:btree_index"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:btree_index"}` — returns the wire type.
- `EXISTS`: `{"key":"example:btree_index"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:btree_index","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:btree_index"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:btree_index"}` — removes the TTL.
- `DEL`: `{"keys":["example:btree_index"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
