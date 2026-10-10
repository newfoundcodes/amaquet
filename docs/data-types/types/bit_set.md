---
title: "bit_set"
---

**Category:** Bit structures  
**Wire type:** `bit_set`  
**Construction:** `CREATE`

Bit structures provide compact bit-level and integer-set storage.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:bit_set","type":"bit_set","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`SET`, `GETBIT`, `COUNT`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:bit_set` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:bit_set",
  "type": "bit_set",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts unsigned `bit`, optional `value:true` and returns previous bit value; storage grows to the addressed 64-bit word.

```json
{
  "key": "example:bit_set",
  "operation": "SET",
  "args": {
    "bit": 7,
    "value": true
  }
}
```

#### `OP GETBIT`

Accepts unsigned `bit` and returns boolean; out-of-range reads are false.

```json
{
  "key": "example:bit_set",
  "operation": "GETBIT",
  "args": {
    "bit": 7
  }
}
```

#### `OP COUNT`

Takes no arguments and returns population count.

```json
{
  "key": "example:bit_set",
  "operation": "COUNT",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:bit_set"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:bit_set"}` — returns the wire type.
- `EXISTS`: `{"key":"example:bit_set"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:bit_set","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:bit_set"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:bit_set"}` — removes the TTL.
- `DEL`: `{"keys":["example:bit_set"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
