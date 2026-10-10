---
title: "decimal"
---

**Category:** Scalar  
**Wire type:** `decimal`  
**Construction:** `SET`

Scalar values are stored directly in a key and use `SET`.

## Create a value

Use the top-level `SET` command to store this directly encoded value. The example shows the required wire representation.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:decimal","type":"decimal","value":"12345.6789"}'
```

## Construction options

This type is constructed with `SET`; it has no `CREATE` options.

## Supported operations

`ADD`, `SUB`, `CMP`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:decimal` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `SET`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:decimal",
  "type": "decimal",
  "value": "12345.6789"
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `value` decimal string and returns exact result string; scales are aligned before arithmetic.

```json
{
  "key": "example:decimal",
  "operation": "ADD",
  "args": {
    "value": "0.3211"
  }
}
```

#### `OP CMP`

Accepts `value` decimal string and returns `-1`, `0`, or `1`; read-only.

```json
{
  "key": "example:decimal",
  "operation": "CMP",
  "args": {
    "value": "12346.0000"
  }
}
```

#### `OP SUB`

Accepts a decimal string and returns the exact result string; scales are aligned before subtraction.

```json
{
  "key": "example:decimal",
  "operation": "SUB",
  "args": {
    "value": "0.3211"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:decimal"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:decimal"}` — returns the wire type.
- `EXISTS`: `{"key":"example:decimal"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:decimal","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:decimal"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:decimal"}` — removes the TTL.
- `DEL`: `{"keys":["example:decimal"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
