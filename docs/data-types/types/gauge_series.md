---
title: "gauge_series"
---

**Category:** Time series  
**Wire type:** `gauge_series`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:gauge_series","type":"gauge_series","options":{"labels":{"metric":"cpu"}}}'
```

## Construction options

Example `CREATE` options: `{"labels":{"metric":"cpu"}}`. Omitted fields use server defaults.

## Supported operations

`SET`, `RANGE`, `LAST`, `AGGREGATE`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:gauge_series` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:gauge_series",
  "type": "gauge_series",
  "options": {
    "labels": {
      "metric": "cpu"
    }
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts optional `timestamp`, numeric `value` and returns `true`; stores the supplied value.

```json
{
  "key": "example:gauge_series",
  "operation": "SET",
  "args": {
    "timestamp": 1000,
    "value": 42.5
  }
}
```

#### `OP LAST`

Takes no arguments and returns last point or `null`.

```json
{
  "key": "example:gauge_series",
  "operation": "LAST",
  "args": {}
}
```

#### `OP RANGE`

Accepts optional Unix-millisecond `from` and `to` bounds and returns inclusive sorted points.

```json
{
  "key": "example:gauge_series",
  "operation": "RANGE",
  "args": {
    "from": 0,
    "to": 3000
  }
}
```

#### `OP AGGREGATE`

Accepts optional bounds, `bucket_ms`, and aggregation mode and returns bucket-start/value points.

```json
{
  "key": "example:gauge_series",
  "operation": "AGGREGATE",
  "args": {
    "from": 0,
    "to": 3000,
    "bucket_ms": 1000,
    "aggregation": "avg"
  }
}
```

#### `OP LEN`

Takes no arguments and returns the point count.

```json
{
  "key": "example:gauge_series",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:gauge_series"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:gauge_series"}` — returns the wire type.
- `EXISTS`: `{"key":"example:gauge_series"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:gauge_series","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:gauge_series"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:gauge_series"}` — removes the TTL.
- `DEL`: `{"keys":["example:gauge_series"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
