---
title: "priority_queue"
---

**Category:** Queues and messaging  
**Wire type:** `priority_queue`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:priority_queue","type":"priority_queue","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ENQUEUE`, `DEQUEUE`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:priority_queue` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:priority_queue",
  "type": "priority_queue",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ENQUEUE`

Accepts nested `value`, numeric `priority` and returns new length.

```json
{
  "key": "example:priority_queue",
  "operation": "ENQUEUE",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "low"
    },
    "priority": 1
  }
}
```

#### `OP LEN`

Takes no arguments and returns length.

```json
{
  "key": "example:priority_queue",
  "operation": "LEN",
  "args": {}
}
```

#### `OP DEQUEUE`

Takes no arguments and returns `{value,priority,sequence}` or `null`; higher priority first, FIFO for ties.

```json
{
  "key": "example:priority_queue",
  "operation": "DEQUEUE",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:priority_queue"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:priority_queue"}` — returns the wire type.
- `EXISTS`: `{"key":"example:priority_queue"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:priority_queue","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:priority_queue"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:priority_queue"}` — removes the TTL.
- `DEL`: `{"keys":["example:priority_queue"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
