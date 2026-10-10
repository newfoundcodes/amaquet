---
title: "persistent_topic"
---

**Category:** Queues and messaging  
**Wire type:** `persistent_topic`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:persistent_topic","type":"persistent_topic","options":{"max_messages":100}}'
```

## Construction options

Example `CREATE` options: `{"max_messages":100}`. Omitted fields use server defaults.

## Supported operations

`PUBLISH`, `READ`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:persistent_topic` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:persistent_topic",
  "type": "persistent_topic",
  "options": {
    "max_messages": 100
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP PUBLISH`

Accepts optional `message_key`, nested `value` and returns monotonic zero-based offset.

```json
{
  "key": "example:persistent_topic",
  "operation": "PUBLISH",
  "args": {
    "message_key": "m1",
    "value": {
      "type": "utf8_string",
      "value": "hello"
    }
  }
}
```

#### `OP READ`

Accepts required `from`, optional `count:100` and returns messages with offset ≥ `from`.

```json
{
  "key": "example:persistent_topic",
  "operation": "READ",
  "args": {
    "from": 0,
    "count": 10
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:persistent_topic"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:persistent_topic"}` — returns the wire type.
- `EXISTS`: `{"key":"example:persistent_topic"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:persistent_topic","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:persistent_topic"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:persistent_topic"}` — removes the TTL.
- `DEL`: `{"keys":["example:persistent_topic"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
