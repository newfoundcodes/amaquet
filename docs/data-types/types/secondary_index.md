---
title: "secondary_index"
---

**Category:** Indexes and search  
**Wire type:** `secondary_index`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:secondary_index","type":"secondary_index","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `LOOKUP`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:secondary_index` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:secondary_index",
  "type": "secondary_index",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `field`, string `value`, `id` and returns `true`.

```json
{
  "key": "example:secondary_index",
  "operation": "ADD",
  "args": {
    "field": "status",
    "value": "active",
    "id": "u1"
  }
}
```

#### `OP LOOKUP`

Accepts `field`, string `value` and returns sorted IDs.

```json
{
  "key": "example:secondary_index",
  "operation": "LOOKUP",
  "args": {
    "field": "status",
    "value": "active"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:secondary_index"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:secondary_index"}` — returns the wire type.
- `EXISTS`: `{"key":"example:secondary_index"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:secondary_index","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:secondary_index"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:secondary_index"}` — removes the TTL.
- `DEL`: `{"keys":["example:secondary_index"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
