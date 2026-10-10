---
title: "numeric_field"
---

**Category:** Indexes and search  
**Wire type:** `numeric_field`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:numeric_field","type":"numeric_field","options":{"name":"score","value":42.5}}'
```

## Construction options

Example `CREATE` options: `{"name":"score","value":42.5}`. Omitted fields use server defaults.

## Supported operations

`SET`, `GET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:numeric_field` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:numeric_field",
  "type": "numeric_field",
  "options": {
    "name": "score",
    "value": 42.5
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts optional `name`, optional numeric `value` and returns `true`.

```json
{
  "key": "example:numeric_field",
  "operation": "SET",
  "args": {
    "name": "score",
    "value": 99.5
  }
}
```

#### `OP GET`

Takes no arguments and returns field object.

```json
{
  "key": "example:numeric_field",
  "operation": "GET",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:numeric_field"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:numeric_field"}` — returns the wire type.
- `EXISTS`: `{"key":"example:numeric_field"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:numeric_field","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:numeric_field"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:numeric_field"}` — removes the TTL.
- `DEL`: `{"keys":["example:numeric_field"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
