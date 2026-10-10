---
title: "multimap"
---

**Category:** Collections  
**Wire type:** `multimap`  
**Construction:** `CREATE`

Collections are synchronized mutable structures manipulated with `OP`.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:multimap","type":"multimap","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `GET`, `REMOVE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:multimap` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:multimap",
  "type": "multimap",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `field`, nested `value` and returns `true`; duplicates are allowed.

```json
{
  "key": "example:multimap",
  "operation": "ADD",
  "args": {
    "field": "tag",
    "value": {
      "type": "utf8_string",
      "value": "alpha"
    }
  }
}
```

#### `OP GET`

Accepts `field` and returns copy of all values.

```json
{
  "key": "example:multimap",
  "operation": "GET",
  "args": {
    "field": "tag"
  }
}
```

#### `OP REMOVE`

Accepts `field` and nested `value`; removes the first equal value and returns whether it was removed.

```json
{
  "key": "example:multimap",
  "operation": "REMOVE",
  "args": {
    "field": "tag",
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
  "key": "example:multimap"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:multimap"}` — returns the wire type.
- `EXISTS`: `{"key":"example:multimap"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:multimap","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:multimap"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:multimap"}` — removes the TTL.
- `DEL`: `{"keys":["example:multimap"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
