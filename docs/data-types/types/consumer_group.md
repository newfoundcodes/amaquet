---
title: "consumer_group"
---

**Category:** Queues and messaging  
**Wire type:** `consumer_group`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:consumer_group","type":"consumer_group","options":{"name":"workers"}}'
```

## Construction options

Example `CREATE` options: `{"name":"workers"}`. Omitted fields use server defaults.

## Supported operations

`SNAPSHOT`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:consumer_group` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:consumer_group",
  "type": "consumer_group",
  "options": {
    "name": "workers"
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SNAPSHOT`

Takes no arguments and returns standalone group's name, cursor, and pending entries.

```json
{
  "key": "example:consumer_group",
  "operation": "SNAPSHOT",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:consumer_group"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:consumer_group"}` — returns the wire type.
- `EXISTS`: `{"key":"example:consumer_group"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:consumer_group","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:consumer_group"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:consumer_group"}` — removes the TTL.
- `DEL`: `{"keys":["example:consumer_group"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
