---
title: "set"
---

**Category:** Collections  
**Wire type:** `set`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:set","type":"set","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `CONTAINS`, `MEMBERS`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:set` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:set",
  "type": "set",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts nested `value` and returns boolean added/removed/present.

```json
{
  "key": "example:set",
  "operation": "ADD",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP CONTAINS`

Accepts nested `value` and returns boolean added/removed/present.

```json
{
  "key": "example:set",
  "operation": "CONTAINS",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP REMOVE`

Accepts nested `value` and returns whether the member existed.

```json
{
  "key": "example:set",
  "operation": "REMOVE",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP MEMBERS`

Takes no arguments and returns members in unspecified order.

```json
{
  "key": "example:set",
  "operation": "MEMBERS",
  "args": {}
}
```

#### `OP LEN`

Takes no arguments and returns the member count.

```json
{
  "key": "example:set",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:set"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:set"}` — returns the wire type.
- `EXISTS`: `{"key":"example:set"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:set","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:set"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:set"}` — removes the TTL.
- `DEL`: `{"keys":["example:set"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
