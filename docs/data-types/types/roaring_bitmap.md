---
title: "roaring_bitmap"
---

**Category:** Bit structures  
**Wire type:** `roaring_bitmap`  
**Construction:** `CREATE`

Bit structures provide compact bit-level and integer-set storage.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:roaring_bitmap","type":"roaring_bitmap","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `CONTAINS`, `VALUES`, `CARDINALITY`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:roaring_bitmap` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:roaring_bitmap",
  "type": "roaring_bitmap",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts unsigned `value` ≤ 2³²−1 and returns boolean.

```json
{
  "key": "example:roaring_bitmap",
  "operation": "ADD",
  "args": {
    "value": 42
  }
}
```

#### `OP CONTAINS`

Accepts unsigned `value` ≤ 2³²−1 and returns boolean.

```json
{
  "key": "example:roaring_bitmap",
  "operation": "CONTAINS",
  "args": {
    "value": 42
  }
}
```

#### `OP CARDINALITY`

Takes no arguments and returns count.

```json
{
  "key": "example:roaring_bitmap",
  "operation": "CARDINALITY",
  "args": {}
}
```

#### `OP REMOVE`

Accepts an unsigned `value` and returns whether it was present.

```json
{
  "key": "example:roaring_bitmap",
  "operation": "REMOVE",
  "args": {
    "value": 42
  }
}
```

#### `OP VALUES`

Takes no arguments and returns sorted uint32 values.

```json
{
  "key": "example:roaring_bitmap",
  "operation": "VALUES",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:roaring_bitmap"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:roaring_bitmap"}` — returns the wire type.
- `EXISTS`: `{"key":"example:roaring_bitmap"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:roaring_bitmap","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:roaring_bitmap"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:roaring_bitmap"}` — removes the TTL.
- `DEL`: `{"keys":["example:roaring_bitmap"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
