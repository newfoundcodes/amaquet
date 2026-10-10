---
title: "timeseries_labels"
---

**Category:** Time series  
**Wire type:** `timeseries_labels`  
**Construction:** `CREATE`

Time-series types store timestamped numeric data and derived statistics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:timeseries_labels","type":"timeseries_labels","options":{"labels":{"env":"prod","region":"local"}}}'
```

## Construction options

Example `CREATE` options: `{"labels":{"env":"prod","region":"local"}}`. Omitted fields use server defaults.

## Supported operations

`GET`, `SET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:timeseries_labels` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:timeseries_labels",
  "type": "timeseries_labels",
  "options": {
    "labels": {
      "env": "prod",
      "region": "local"
    }
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP SET`

Accepts `label`, string `value` and returns `true`.

```json
{
  "key": "example:timeseries_labels",
  "operation": "SET",
  "args": {
    "label": "region",
    "value": "ph"
  }
}
```

#### `OP GET`

Takes no arguments and returns copy of label map.

```json
{
  "key": "example:timeseries_labels",
  "operation": "GET",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:timeseries_labels"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:timeseries_labels"}` — returns the wire type.
- `EXISTS`: `{"key":"example:timeseries_labels"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:timeseries_labels","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:timeseries_labels"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:timeseries_labels"}` — removes the TTL.
- `DEL`: `{"keys":["example:timeseries_labels"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
