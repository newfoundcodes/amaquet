---
title: "array"
---

**Category:** Collections  
**Wire type:** `array`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:array","type":"array","options":{"typed":true,"element_type":"utf8_string"}}'
```

## Construction options

Example `CREATE` options: `{"typed":true,"element_type":"utf8_string"}`. Omitted fields use server defaults.

## Supported operations

`APPEND`, `SET`, `GET`, `VALUES`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:array` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:array",
  "type": "array",
  "options": {
    "typed": true,
    "element_type": "utf8_string"
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP APPEND`

Accepts nested `value` and returns new length; enforces typed-array element type.

```json
{
  "key": "example:array",
  "operation": "APPEND",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP GET`

Accepts `index` and returns typed value or `null`; negative indexes allowed.

```json
{
  "key": "example:array",
  "operation": "GET",
  "args": {
    "index": 0
  }
}
```

#### `OP LEN`

Takes no arguments and returns the array length.

```json
{
  "key": "example:array",
  "operation": "LEN",
  "args": {}
}
```

#### `OP SET`

Accepts `index` and nested `value` and returns `true`; negative indexes are allowed and the element type is enforced when the array is typed.

```json
{
  "key": "example:array",
  "operation": "SET",
  "args": {
    "index": 0,
    "value": {
      "type": "utf8_string",
      "value": "updated"
    }
  }
}
```

#### `OP VALUES`

Takes no arguments and returns a copy of the array values.

```json
{
  "key": "example:array",
  "operation": "VALUES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:array"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:array"}` — returns the wire type.
- `EXISTS`: `{"key":"example:array"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:array","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:array"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:array"}` — removes the TTL.
- `DEL`: `{"keys":["example:array"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
