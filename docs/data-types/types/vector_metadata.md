---
title: "vector_metadata"
---

**Category:** Vectors  
**Wire type:** `vector_metadata`  
**Construction:** `CREATE`

Vector types support embeddings, metadata, quantization, and similarity search.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:vector_metadata","type":"vector_metadata","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`GET`, `SET`, `DELETE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:vector_metadata` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:vector_metadata",
  "type": "vector_metadata",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts `field`, arbitrary JSON `value` and returns `true`.

```json
{
  "key": "example:vector_metadata",
  "operation": "SET",
  "args": {
    "field": "lang",
    "value": "en"
  }
}
```

#### `OP GET`

Takes no arguments and returns copy of metadata map.

```json
{
  "key": "example:vector_metadata",
  "operation": "GET",
  "args": {}
}
```

#### `OP DELETE`

Accepts `field` and returns whether the field existed.

```json
{
  "key": "example:vector_metadata",
  "operation": "DELETE",
  "args": {
    "field": "lang"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:vector_metadata"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:vector_metadata"}` — returns the wire type.
- `EXISTS`: `{"key":"example:vector_metadata"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:vector_metadata","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:vector_metadata"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:vector_metadata"}` — removes the TTL.
- `DEL`: `{"keys":["example:vector_metadata"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
