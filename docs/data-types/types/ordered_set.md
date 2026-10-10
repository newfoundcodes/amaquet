---
title: "ordered_set"
---

**Category:** Collections  
**Wire type:** `ordered_set`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:ordered_set","type":"ordered_set","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `VALUES`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:ordered_set` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:ordered_set",
  "type": "ordered_set",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts nested `value` and returns boolean; re-adding an existing value does not move it.

```json
{
  "key": "example:ordered_set",
  "operation": "ADD",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP VALUES`

Takes no arguments and returns insertion-order values.

```json
{
  "key": "example:ordered_set",
  "operation": "VALUES",
  "args": {}
}
```

#### `OP REMOVE`

Accepts nested `value` and returns whether it was removed.

```json
{
  "key": "example:ordered_set",
  "operation": "REMOVE",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:ordered_set"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:ordered_set"}` — returns the wire type.
- `EXISTS`: `{"key":"example:ordered_set"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:ordered_set","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:ordered_set"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:ordered_set"}` — removes the TTL.
- `DEL`: `{"keys":["example:ordered_set"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
