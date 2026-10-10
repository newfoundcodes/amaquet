---
title: "timeseries"
---

**Category:** Time series  
**Wire type:** `timeseries`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:timeseries","type":"timeseries","options":{"labels":{"service":"api","env":"prod"},"retention_ms":3600000}}'
```

## Construction options

Example `CREATE` options: `{"labels":{"service":"api","env":"prod"},"retention_ms":3600000}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `RANGE`, `LAST`, `AGGREGATE`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:timeseries` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:timeseries",
  "type": "timeseries",
  "options": {
    "labels": {
      "service": "api",
      "env": "prod"
    },
    "retention_ms": 3600000
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts optional `timestamp` Unix ms (default now), numeric `value` and returns `true`; replaces an existing point at the same timestamp.

```json
{
  "key": "example:timeseries",
  "operation": "ADD",
  "args": {
    "timestamp": 1730000000000,
    "value": 10
  }
}
```

#### `OP LEN`

Takes no arguments and returns point count.

```json
{
  "key": "example:timeseries",
  "operation": "LEN",
  "args": {}
}
```

#### `OP RANGE`

Accepts optional Unix-millisecond `from` and `to` bounds and returns inclusive sorted points.

```json
{
  "key": "example:timeseries",
  "operation": "RANGE",
  "args": {
    "from": 0,
    "to": 3000
  }
}
```

#### `OP LAST`

Takes no arguments and returns the last point or `null`.

```json
{
  "key": "example:timeseries",
  "operation": "LAST",
  "args": {}
}
```

#### `OP AGGREGATE`

Accepts optional bounds, `bucket_ms`, and aggregation mode and returns bucket-start/value points.

```json
{
  "key": "example:timeseries",
  "operation": "AGGREGATE",
  "args": {
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
  "key": "example:timeseries"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:timeseries"}` — returns the wire type.
- `EXISTS`: `{"key":"example:timeseries"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:timeseries","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:timeseries"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:timeseries"}` — removes the TTL.
- `DEL`: `{"keys":["example:timeseries"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
