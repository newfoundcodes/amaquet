---
title: "tuple"
---

**Category:** Collections  
**Wire type:** `tuple`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:tuple","type":"tuple","options":{"values":[{"type":"utf8_string","value":"alpha"},{"type":"integer","value":7}]}}'
```

## Construction options

Example `CREATE` options: `{"values":[{"type":"utf8_string","value":"alpha"},{"type":"integer","value":7}]}`. Omitted fields use server defaults.

## Supported operations

`GET`, `VALUES`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:tuple` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:tuple",
  "type": "tuple",
  "options": {
    "values": [
      {
        "type": "utf8_string",
        "value": "alpha"
      },
      {
        "type": "integer",
        "value": 7
      }
    ]
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP LEN`

Takes no arguments and returns the tuple length.

```json
{
  "key": "example:tuple",
  "operation": "LEN",
  "args": {}
}
```

#### `OP GET`

Accepts `index` and returns typed value or `null`; negative indexes allowed.

```json
{
  "key": "example:tuple",
  "operation": "GET",
  "args": {
    "index": 0
  }
}
```

#### `OP VALUES`

Takes no arguments and returns the fixed tuple values.

```json
{
  "key": "example:tuple",
  "operation": "VALUES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:tuple"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:tuple"}` — returns the wire type.
- `EXISTS`: `{"key":"example:tuple"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:tuple","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:tuple"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:tuple"}` — removes the TTL.
- `DEL`: `{"keys":["example:tuple"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
