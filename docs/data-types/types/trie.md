---
title: "trie"
---

**Category:** Indexes and search  
**Wire type:** `trie`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:trie","type":"trie","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`INSERT`, `EXACT`, `PREFIX`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:trie` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:trie",
  "type": "trie",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP INSERT`

Accepts string `value`, `id` and returns `true`.

```json
{
  "key": "example:trie",
  "operation": "INSERT",
  "args": {
    "value": "amaquet",
    "id": "1"
  }
}
```

#### `OP PREFIX`

Accepts string `prefix`, optional `limit:100` and returns lexically traversed map of matching strings to IDs.

```json
{
  "key": "example:trie",
  "operation": "PREFIX",
  "args": {
    "prefix": "ama",
    "limit": 10
  }
}
```

#### `OP EXACT`

Accepts string `value` and returns sorted IDs for an exact key.

```json
{
  "key": "example:trie",
  "operation": "EXACT",
  "args": {
    "value": "amaquet"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:trie"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:trie"}` — returns the wire type.
- `EXISTS`: `{"key":"example:trie"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:trie","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:trie"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:trie"}` — removes the TTL.
- `DEL`: `{"keys":["example:trie"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
