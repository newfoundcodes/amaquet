---
title: "histogram"
---

**Category:** Time series  
**Wire type:** `histogram`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:histogram","type":"histogram","options":{"bounds":[1,5,10,50,100]}}'
```

## Construction options

Example `CREATE` options: `{"bounds":[1,5,10,50,100]}`. Omitted fields use server defaults.

## Supported operations

`OBSERVE`, `SNAPSHOT`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:histogram` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:histogram",
  "type": "histogram",
  "options": {
    "bounds": [
      1,
      5,
      10,
      50,
      100
    ]
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP OBSERVE`

Accepts numeric `value` and returns `true`; increments first bound ≥ value, otherwise overflow bucket.

```json
{
  "key": "example:histogram",
  "operation": "OBSERVE",
  "args": {
    "value": 3
  }
}
```

#### `OP SNAPSHOT`

Takes no arguments and returns `{bounds,counts,count,sum}`.

```json
{
  "key": "example:histogram",
  "operation": "SNAPSHOT",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:histogram"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:histogram"}` — returns the wire type.
- `EXISTS`: `{"key":"example:histogram"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:histogram","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:histogram"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:histogram"}` — removes the TTL.
- `DEL`: `{"keys":["example:histogram"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
