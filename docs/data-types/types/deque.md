---
title: "deque"
---

**Category:** Collections  
**Wire type:** `deque`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:deque","type":"deque","options":{"capacity":8}}'
```

## Construction options

Example `CREATE` options: `{"capacity":8}`. Omitted fields use server defaults.

## Supported operations

`PUSH_FRONT`, `PUSH_BACK`, `POP_FRONT`, `POP_BACK`, `VALUES`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:deque` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:deque",
  "type": "deque",
  "options": {
    "capacity": 8
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP PUSH_BACK`

Accepts nested `value` and returns new length.

```json
{
  "key": "example:deque",
  "operation": "PUSH_BACK",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP PUSH_FRONT`

Accepts nested `value` and returns new length.

```json
{
  "key": "example:deque",
  "operation": "PUSH_FRONT",
  "args": {
    "value": {
      "type": "integer",
      "value": 7
    }
  }
}
```

#### `OP LEN`

Takes no arguments and returns logical-order values or length.

```json
{
  "key": "example:deque",
  "operation": "LEN",
  "args": {}
}
```

#### `OP POP_FRONT`

Takes no arguments and returns the first typed value or `null`.

```json
{
  "key": "example:deque",
  "operation": "POP_FRONT",
  "args": {}
}
```

#### `OP POP_BACK`

Takes no arguments and returns the last typed value or `null`.

```json
{
  "key": "example:deque",
  "operation": "POP_BACK",
  "args": {}
}
```

#### `OP VALUES`

Takes no arguments and returns values in logical order.

```json
{
  "key": "example:deque",
  "operation": "VALUES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:deque"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:deque"}` — returns the wire type.
- `EXISTS`: `{"key":"example:deque"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:deque","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:deque"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:deque"}` — removes the TTL.
- `DEL`: `{"keys":["example:deque"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
