---
title: "ring_buffer"
---

**Category:** Collections  
**Wire type:** `ring_buffer`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:ring_buffer","type":"ring_buffer","options":{"capacity":4}}'
```

## Construction options

Example `CREATE` options: `{"capacity":4}`. Omitted fields use server defaults.

## Supported operations

`PUSH`, `VALUES`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:ring_buffer` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:ring_buffer",
  "type": "ring_buffer",
  "options": {
    "capacity": 4
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP PUSH`

Accepts nested `value` and returns current length; overwrites oldest at capacity.

```json
{
  "key": "example:ring_buffer",
  "operation": "PUSH",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "a"
    }
  }
}
```

#### `OP LEN`

Takes no arguments and returns oldest-to-newest values or length.

```json
{
  "key": "example:ring_buffer",
  "operation": "LEN",
  "args": {}
}
```

#### `OP VALUES`

Takes no arguments and returns values from oldest to newest.

```json
{
  "key": "example:ring_buffer",
  "operation": "VALUES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:ring_buffer"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:ring_buffer"}` — returns the wire type.
- `EXISTS`: `{"key":"example:ring_buffer"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:ring_buffer","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:ring_buffer"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:ring_buffer"}` — removes the TTL.
- `DEL`: `{"keys":["example:ring_buffer"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
