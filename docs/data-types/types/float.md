---
title: "float"
---

**Category:** Scalar  
**Wire type:** `float`  
**Construction:** `SET`

Scalar values are stored directly in a key and use `SET`.

## Create a value

Use the top-level `SET` command to store this directly encoded value. The example shows the required wire representation.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:float","type":"float","value":3.14159}'
```

## Construction options

This type is constructed with `SET`; it has no `CREATE` options.

## Supported operations

`ADD`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:float` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `SET`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:float",
  "type": "float",
  "value": 3.14159
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `delta` number and returns new float64 value.

```json
{
  "key": "example:float",
  "operation": "ADD",
  "args": {
    "delta": 0.5
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:float"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:float"}` — returns the wire type.
- `EXISTS`: `{"key":"example:float"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:float","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:float"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:float"}` — removes the TTL.
- `DEL`: `{"keys":["example:float"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
