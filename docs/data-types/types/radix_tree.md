---
title: "radix_tree"
---

**Category:** Indexes and search  
**Wire type:** `radix_tree`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:radix_tree","type":"radix_tree","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`INSERT`, `EXACT`, `PREFIX`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:radix_tree` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:radix_tree",
  "type": "radix_tree",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP INSERT`

Accepts string `value`, `id` and returns `true`.

```json
{
  "key": "example:radix_tree",
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
  "key": "example:radix_tree",
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
  "key": "example:radix_tree",
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
  "key": "example:radix_tree"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:radix_tree"}` — returns the wire type.
- `EXISTS`: `{"key":"example:radix_tree"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:radix_tree","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:radix_tree"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:radix_tree"}` — removes the TTL.
- `DEL`: `{"keys":["example:radix_tree"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
