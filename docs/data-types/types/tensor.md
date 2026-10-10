---
title: "tensor"
---

**Category:** Structured  
**Wire type:** `tensor`  
**Construction:** `CREATE`

Structured values cover documents, encoded payloads, records, matrices, and tensors.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:tensor","type":"tensor","options":{"shape":[2,2,2],"dtype":"float64","data":[0,1,2,3,4,5,6,7]}}'
```

## Construction options

Example `CREATE` options: `{"shape":[2,2,2],"dtype":"float64","data":[0,1,2,3,4,5,6,7]}`. Omitted fields use server defaults.

## Supported operations

`GET`, `SET`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:tensor` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:tensor",
  "type": "tensor",
  "options": {
    "shape": [
      2,
      2,
      2
    ],
    "dtype": "float64",
    "data": [
      0,
      1,
      2,
      3,
      4,
      5,
      6,
      7
    ]
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP GET`

Accepts `indices:[...]` and returns numeric cell; rank and bounds enforced.

```json
{
  "key": "example:tensor",
  "operation": "GET",
  "args": {
    "indices": [
      1,
      1,
      1
    ]
  }
}
```

#### `OP SET`

Accepts `indices:[...]`, numeric `value` and returns `true`.

```json
{
  "key": "example:tensor",
  "operation": "SET",
  "args": {
    "indices": [
      0,
      0,
      0
    ],
    "value": 99
  }
}
```

#### `OP LEN`

Takes no arguments and returns the flat element count.

```json
{
  "key": "example:tensor",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:tensor"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:tensor"}` — returns the wire type.
- `EXISTS`: `{"key":"example:tensor"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:tensor","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:tensor"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:tensor"}` — removes the TTL.
- `DEL`: `{"keys":["example:tensor"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
