---
title: "stream"
---

**Category:** Queues and messaging  
**Wire type:** `stream`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:stream","type":"stream","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `RANGE`, `GROUP_CREATE`, `GROUP_READ`, `ACK`, `PENDING`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:stream` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:stream",
  "type": "stream",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `fields` map of nested values; optional `id:"millis-sequence"` and returns monotonic stream ID; server generates a deterministic ID when omitted.

```json
{
  "key": "example:stream",
  "operation": "ADD",
  "args": {
    "fields": {
      "msg": {
        "type": "utf8_string",
        "value": "hello"
      }
    }
  }
}
```

#### `OP RANGE`

Accepts optional `count:100` and returns first entries up to count; current protocol does not expose start/end bounds.

```json
{
  "key": "example:stream",
  "operation": "RANGE",
  "args": {
    "count": 10
  }
}
```

#### `OP GROUP_CREATE`

Accepts `group` and returns `true`; starts before the first entry.

```json
{
  "key": "example:stream",
  "operation": "GROUP_CREATE",
  "args": {
    "group": "g1"
  }
}
```

#### `OP GROUP_READ`

Accepts `group`, `consumer`, and optional `count`; returns new entries and records them pending.

```json
{
  "key": "example:stream",
  "operation": "GROUP_READ",
  "args": {
    "group": "workers",
    "consumer": "c1",
    "count": 10
  }
}
```

#### `OP ACK`

Accepts `group` and stream `ids`; returns the number removed from pending.

```json
{
  "key": "example:stream",
  "operation": "ACK",
  "args": {
    "group": "workers",
    "ids": [
      "0-1"
    ]
  }
}
```

#### `OP PENDING`

Accepts `group` and returns pending records sorted by stream ID.

```json
{
  "key": "example:stream",
  "operation": "PENDING",
  "args": {
    "group": "workers"
  }
}
```

#### `OP LEN`

Takes no arguments and returns the entry count.

```json
{
  "key": "example:stream",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:stream"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:stream"}` — returns the wire type.
- `EXISTS`: `{"key":"example:stream"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:stream","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:stream"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:stream"}` — removes the TTL.
- `DEL`: `{"keys":["example:stream"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
