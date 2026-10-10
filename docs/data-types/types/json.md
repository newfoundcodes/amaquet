---
title: "json"
---

**Category:** Structured  
**Wire type:** `json`  
**Construction:** `CREATE` (also supports top-level `SET`)

Structured values cover documents, encoded payloads, records, matrices, and tensors.

## Create a value

Use `CREATE` to initialize an empty JSON object, or use top-level `SET` to store a complete JSON value directly.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:json","type":"json","options":{}}'
```

For direct initialization, supply the document as the `value` field:

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:json","type":"json","value":{"ready":true}}'
```

## Construction options

`CREATE` accepts no type-specific options and starts with `{}`. `SET` accepts any valid JSON value and preserves JSON number tokens during decoding.

## Supported operations

`GET`, `SET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:json` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:json",
  "type": "json",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts arbitrary JSON `value` and returns `true`; replaces the whole document.

```json
{
  "key": "example:json",
  "operation": "SET",
  "args": {
    "value": {
      "hello": "world",
      "n": 1
    }
  }
}
```

#### `OP GET`

Takes no arguments and returns deep-copied JSON document.

```json
{
  "key": "example:json",
  "operation": "GET",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:json"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:json"}` — returns the wire type.
- `EXISTS`: `{"key":"example:json"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:json","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:json"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:json"}` — removes the TTL.
- `DEL`: `{"keys":["example:json"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
