---
title: "bit_field"
---

**Category:** Bit structures  
**Wire type:** `bit_field`  
**Construction:** `CREATE`

Bit structures provide compact bit-level and integer-set storage.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:bit_field","type":"bit_field","options":{"size_bytes":8}}'
```

## Construction options

Example `CREATE` options: `{"size_bytes":8}`. Omitted fields use server defaults.

## Supported operations

`SET`, `GET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:bit_field` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:bit_field",
  "type": "bit_field",
  "options": {
    "size_bytes": 8
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts `offset`, `width`, `value` unsigned and returns `true`; width must be 1…64; bits are big-endian within bytes.

```json
{
  "key": "example:bit_field",
  "operation": "SET",
  "args": {
    "offset": 4,
    "width": 8,
    "value": 171
  }
}
```

#### `OP GET`

Accepts `offset`, `width` and returns unsigned value; an unread allocated range returns 0.

```json
{
  "key": "example:bit_field",
  "operation": "GET",
  "args": {
    "offset": 4,
    "width": 8
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:bit_field"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:bit_field"}` — returns the wire type.
- `EXISTS`: `{"key":"example:bit_field"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:bit_field","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:bit_field"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:bit_field"}` — removes the TTL.
- `DEL`: `{"keys":["example:bit_field"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
