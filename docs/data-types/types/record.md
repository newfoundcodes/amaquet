---
title: "record"
---

**Category:** Structured  
**Wire type:** `record`  
**Construction:** `CREATE`

Structured values cover documents, encoded payloads, records, matrices, and tensors.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:record","type":"record","options":{"schema":[{"name":"name","type":"utf8_string","nullable":false},{"name":"age","type":"integer","nullable":true}]}}'
```

## Construction options

Example `CREATE` options: `{"schema":[{"name":"name","type":"utf8_string","nullable":false},{"name":"age","type":"integer","nullable":true}]}`. Omitted fields use server defaults.

## Supported operations

`SET`, `GET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:record` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:record",
  "type": "record",
  "options": {
    "schema": [
      {
        "name": "name",
        "type": "utf8_string",
        "nullable": false
      },
      {
        "name": "age",
        "type": "integer",
        "nullable": true
      }
    ]
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts `field`, nested `value` and returns `true`; schema/type/nullability enforced.

```json
{
  "key": "example:record",
  "operation": "SET",
  "args": {
    "field": "name",
    "value": {
      "type": "utf8_string",
      "value": "alice"
    }
  }
}
```

#### `OP GET`

Accepts `field` and returns typed value or `null`.

```json
{
  "key": "example:record",
  "operation": "GET",
  "args": {
    "field": "name"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:record"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:record"}` — returns the wire type.
- `EXISTS`: `{"key":"example:record"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:record","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:record"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:record"}` — removes the TTL.
- `DEL`: `{"keys":["example:record"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
