---
title: "counting_bloom_filter"
---

**Category:** Probabilistic  
**Wire type:** `counting_bloom_filter`  
**Construction:** `CREATE`

Probabilistic structures provide bounded-memory estimates and membership/frequency queries.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:counting_bloom_filter","type":"counting_bloom_filter","options":{"expected":1000,"false_positive_rate":0.01}}'
```

## Construction options

Example `CREATE` options: `{"expected":1000,"false_positive_rate":0.01}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `CONTAINS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:counting_bloom_filter` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:counting_bloom_filter",
  "type": "counting_bloom_filter",
  "options": {
    "expected": 1000,
    "false_positive_rate": 0.01
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts string `value` and returns `true`; saturating uint16 counters.

```json
{
  "key": "example:counting_bloom_filter",
  "operation": "ADD",
  "args": {
    "value": "alice"
  }
}
```

#### `OP CONTAINS`

Accepts string `value` and returns probabilistic membership.

```json
{
  "key": "example:counting_bloom_filter",
  "operation": "CONTAINS",
  "args": {
    "value": "alice"
  }
}
```

#### `OP REMOVE`

Accepts string `value` and returns a boolean; it decrements counters only when every counter is nonzero.

```json
{
  "key": "example:counting_bloom_filter",
  "operation": "REMOVE",
  "args": {
    "value": "alice"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:counting_bloom_filter"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:counting_bloom_filter"}` — returns the wire type.
- `EXISTS`: `{"key":"example:counting_bloom_filter"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:counting_bloom_filter","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:counting_bloom_filter"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:counting_bloom_filter"}` — removes the TTL.
- `DEL`: `{"keys":["example:counting_bloom_filter"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
