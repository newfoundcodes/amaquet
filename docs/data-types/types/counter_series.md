---
title: "counter_series"
---

**Category:** Time series  
**Wire type:** `counter_series`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:counter_series","type":"counter_series","options":{"labels":{"metric":"requests"}}}'
```

## Construction options

Example `CREATE` options: `{"labels":{"metric":"requests"}}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `RANGE`, `LAST`, `AGGREGATE`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:counter_series` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:counter_series",
  "type": "counter_series",
  "options": {
    "labels": {
      "metric": "requests"
    }
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts optional `timestamp`, numeric `value` delta and returns `true`; stores last value plus delta.

```json
{
  "key": "example:counter_series",
  "operation": "ADD",
  "args": {
    "timestamp": 1000,
    "value": 5
  }
}
```

#### `OP LAST`

Takes no arguments and returns last point or `null`.

```json
{
  "key": "example:counter_series",
  "operation": "LAST",
  "args": {}
}
```

#### `OP AGGREGATE`

Accepts optional `from`, `to`, `bucket_ms`, and `aggregation` and returns bucket-start/value points.

```json
{
  "key": "example:counter_series",
  "operation": "AGGREGATE",
  "args": {
    "from": 0,
    "to": 3000,
    "bucket_ms": 1000,
    "aggregation": "sum"
  }
}
```

#### `OP LEN`

Takes no arguments and returns the point count.

```json
{
  "key": "example:counter_series",
  "operation": "LEN",
  "args": {}
}
```

#### `OP RANGE`

Accepts optional `from` and `to` Unix-millisecond bounds and returns inclusive sorted points.

```json
{
  "key": "example:counter_series",
  "operation": "RANGE",
  "args": {
    "from": 0,
    "to": 3000
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:counter_series"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:counter_series"}` — returns the wire type.
- `EXISTS`: `{"key":"example:counter_series"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:counter_series","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:counter_series"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:counter_series"}` — removes the TTL.
- `DEL`: `{"keys":["example:counter_series"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
