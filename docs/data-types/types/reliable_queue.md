---
title: "reliable_queue"
---

**Category:** Queues and messaging  
**Wire type:** `reliable_queue`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:reliable_queue","type":"reliable_queue","options":{"max_attempts":3}}'
```

## Construction options

Example `CREATE` options: `{"max_attempts":3}`. Omitted fields use server defaults.

## Supported operations

`ENQUEUE`, `CLAIM`, `ACK`, `NACK`, `STATS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:reliable_queue` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:reliable_queue",
  "type": "reliable_queue",
  "options": {
    "max_attempts": 3
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ENQUEUE`

Accepts `id`, nested `value` and returns `true`; IDs are not checked for uniqueness.

```json
{
  "key": "example:reliable_queue",
  "operation": "ENQUEUE",
  "args": {
    "id": "job1",
    "value": {
      "type": "utf8_string",
      "value": "work"
    }
  }
}
```

#### `OP CLAIM`

Accepts optional `lease_ms:30000` and returns oldest ready message or `null`; increments attempts.

```json
{
  "key": "example:reliable_queue",
  "operation": "CLAIM",
  "args": {
    "lease_ms": 5000
  }
}
```

#### `OP ACK`

Accepts `id` and returns boolean; NACK requeues or dead-letters at `max_attempts`.

```json
{
  "key": "example:reliable_queue",
  "operation": "ACK",
  "args": {
    "id": "job1"
  }
}
```

#### `OP NACK`

Accepts `id` and returns whether the message was requeued or dead-lettered.

```json
{
  "key": "example:reliable_queue",
  "operation": "NACK",
  "args": {
    "id": "job1"
  }
}
```

#### `OP STATS`

Takes no arguments and returns ready, pending, and dead counts after expired leases are reconciled.

```json
{
  "key": "example:reliable_queue",
  "operation": "STATS",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:reliable_queue"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:reliable_queue"}` — returns the wire type.
- `EXISTS`: `{"key":"example:reliable_queue"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:reliable_queue","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:reliable_queue"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:reliable_queue"}` — removes the TTL.
- `DEL`: `{"keys":["example:reliable_queue"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
