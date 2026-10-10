---
title: "ordered_map"
---

**Category:** Collections  
**Wire type:** `ordered_map`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:ordered_map","type":"ordered_map","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`SET`, `GET`, `DELETE`, `ENTRIES`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:ordered_map` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:ordered_map",
  "type": "ordered_map",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts `field`, nested `value` and returns `true`; replacement retains original position.

```json
{
  "key": "example:ordered_map",
  "operation": "SET",
  "args": {
    "field": "name",
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP GET`

Accepts `field` and returns the typed value or `null`.

```json
{
  "key": "example:ordered_map",
  "operation": "GET",
  "args": {
    "field": "name"
  }
}
```

#### `OP DELETE`

Accepts `field` and returns whether the field existed.

```json
{
  "key": "example:ordered_map",
  "operation": "DELETE",
  "args": {
    "field": "name"
  }
}
```

#### `OP ENTRIES`

Takes no arguments and returns key/value entries in insertion order.

```json
{
  "key": "example:ordered_map",
  "operation": "ENTRIES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:ordered_map"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:ordered_map"}` — returns the wire type.
- `EXISTS`: `{"key":"example:ordered_map"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:ordered_map","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:ordered_map"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:ordered_map"}` — removes the TTL.
- `DEL`: `{"keys":["example:ordered_map"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
