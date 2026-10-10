---
title: "atomic_counter"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `atomic_counter`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:atomic_counter","type":"atomic_counter","options":{"initial":10}}'
```

## Construction options

Example `CREATE` options: `{"initial":10}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `LOAD`, `CAS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:atomic_counter` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:atomic_counter",
  "type": "atomic_counter",
  "options": {
    "initial": 10
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts signed `delta` and returns new value.

```json
{
  "key": "example:atomic_counter",
  "operation": "ADD",
  "args": {
    "delta": 5
  }
}
```

#### `OP CAS`

Accepts signed `old`, signed `new` and returns swap-success boolean.

```json
{
  "key": "example:atomic_counter",
  "operation": "CAS",
  "args": {
    "old": 15,
    "new": 20
  }
}
```

#### `OP LOAD`

Takes no arguments and returns current value.

```json
{
  "key": "example:atomic_counter",
  "operation": "LOAD",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:atomic_counter"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:atomic_counter"}` — returns the wire type.
- `EXISTS`: `{"key":"example:atomic_counter"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:atomic_counter","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:atomic_counter"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:atomic_counter"}` — removes the TTL.
- `DEL`: `{"keys":["example:atomic_counter"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
