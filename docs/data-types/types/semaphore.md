---
title: "semaphore"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `semaphore`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:semaphore","type":"semaphore","options":{"limit":2}}'
```

## Construction options

Example `CREATE` options: `{"limit":2}`. Omitted fields use server defaults.

## Supported operations

`ACQUIRE`, `TRY_ACQUIRE`, `RELEASE`, `STATS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:semaphore` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:semaphore",
  "type": "semaphore",
  "options": {
    "limit": 2
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP TRY_ACQUIRE`

Takes no arguments and returns immediate boolean.

```json
{
  "key": "example:semaphore",
  "operation": "TRY_ACQUIRE",
  "args": {}
}
```

#### `OP STATS`

Takes no arguments and returns `{limit,in_use}`.

```json
{
  "key": "example:semaphore",
  "operation": "STATS",
  "args": {}
}
```

#### `OP RELEASE`

Takes no arguments and returns boolean; false when no permit is held.

```json
{
  "key": "example:semaphore",
  "operation": "RELEASE",
  "args": {}
}
```

#### `OP ACQUIRE`

Takes no arguments and waits for a permit before returning `true`.

```json
{
  "key": "example:semaphore",
  "operation": "ACQUIRE",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:semaphore"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:semaphore"}` — returns the wire type.
- `EXISTS`: `{"key":"example:semaphore"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:semaphore","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:semaphore"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:semaphore"}` — removes the TTL.
- `DEL`: `{"keys":["example:semaphore"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
