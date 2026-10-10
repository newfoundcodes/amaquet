---
title: "adjacency_set"
---

**Category:** Graph, synchronization, and network  
**Wire type:** `adjacency_set`  
**Construction:** `CREATE`

These types cover graph adjacency, atomic coordination, synchronization, rate limiting, and network values.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:adjacency_set","type":"adjacency_set","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `NEIGHBORS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:adjacency_set` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:adjacency_set",
  "type": "adjacency_set",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `from`, `to`, optional numeric `weight:1` and returns `true`; edges are directed.

```json
{
  "key": "example:adjacency_set",
  "operation": "ADD",
  "args": {
    "from": "a",
    "to": "b",
    "weight": 1.5
  }
}
```

#### `OP NEIGHBORS`

Accepts `from` and returns map of outgoing neighbor to weight.

```json
{
  "key": "example:adjacency_set",
  "operation": "NEIGHBORS",
  "args": {
    "from": "a"
  }
}
```

#### `OP REMOVE`

Accepts `from` and `to` and returns whether the directed edge existed.

```json
{
  "key": "example:adjacency_set",
  "operation": "REMOVE",
  "args": {
    "from": "a",
    "to": "b"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:adjacency_set"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:adjacency_set"}` — returns the wire type.
- `EXISTS`: `{"key":"example:adjacency_set"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:adjacency_set","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:adjacency_set"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:adjacency_set"}` — removes the TTL.
- `DEL`: `{"keys":["example:adjacency_set"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
