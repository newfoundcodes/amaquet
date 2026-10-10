---
title: "aggregated_series"
---

**Category:** Time series  
**Wire type:** `aggregated_series`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:aggregated_series","type":"aggregated_series","options":{"bucket_ms":1000,"aggregation":"avg"}}'
```

## Construction options

Example `CREATE` options: `{"bucket_ms":1000,"aggregation":"avg"}`. Omitted fields use server defaults.

## Supported operations

`RANGE`, `AGGREGATE`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:aggregated_series` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:aggregated_series",
  "type": "aggregated_series",
  "options": {
    "bucket_ms": 1000,
    "aggregation": "avg"
  }
}
```

An aggregated series reads a separate source series. Create that source key independently:

#### Source `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:timeseries:source",
  "type": "timeseries",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP AGGREGATE`

Accepts `source_key`; optional `from`, `to`, `bucket_ms`, and `aggregation`, and returns bucketed points from a `timeseries`, `counter_series`, or `gauge_series` key.

```json
{
  "key": "example:aggregated_series",
  "operation": "AGGREGATE",
  "args": {
    "source_key": "example:timeseries:source",
    "from": 0,
    "to": 3000,
    "bucket_ms": 1000,
    "aggregation": "avg"
  }
}
```

#### `OP RANGE`

Accepts `source_key`; optional `from`, `to`, `bucket_ms`, and `aggregation`, and returns bucketed points from a time-series source.

```json
{
  "key": "example:aggregated_series",
  "operation": "RANGE",
  "args": {
    "source_key": "example:timeseries:source",
    "from": 0,
    "to": 3000,
    "bucket_ms": 1000,
    "aggregation": "avg"
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:aggregated_series"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:aggregated_series"}` — returns the wire type.
- `EXISTS`: `{"key":"example:aggregated_series"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:aggregated_series","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:aggregated_series"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:aggregated_series"}` — removes the TTL.
- `DEL`: `{"keys":["example:aggregated_series"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
