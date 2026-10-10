---
title: "bounding_box"
---

**Category:** Geospatial  
**Wire type:** `bounding_box`  
**Construction:** `SET`

Geospatial types use latitude/longitude coordinates and spatial predicates.

## Create a value

Use the top-level `SET` command to store this directly encoded value. The example shows the required wire representation.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:bounding_box","type":"bounding_box","value":{"MinLat":14.0,"MinLon":120.0,"MaxLat":15.0,"MaxLon":122.0}}'
```

## Construction options

This type is constructed with `SET`; it has no `CREATE` options.

## Supported operations

`CONTAINS`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:bounding_box` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `SET`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:bounding_box",
  "type": "bounding_box",
  "value": {
    "MinLat": 14.0,
    "MinLon": 120.0,
    "MaxLat": 15.0,
    "MaxLon": 122.0
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP CONTAINS`

Accepts `point:{lat,lon}` and returns boolean containment; polygon uses ray casting on lon/lat coordinates.

```json
{
  "key": "example:bounding_box",
  "operation": "CONTAINS",
  "args": {
    "point": {
      "Lat": 14.5,
      "Lon": 121.0
    }
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:bounding_box"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:bounding_box"}` — returns the wire type.
- `EXISTS`: `{"key":"example:bounding_box"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:bounding_box","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:bounding_box"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:bounding_box"}` — removes the TTL.
- `DEL`: `{"keys":["example:bounding_box"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
