---
title: "geo_spatial_index"
---

**Category:** Geospatial  
**Wire type:** `geo_spatial_index`  
**Construction:** `CREATE`

Geospatial types use latitude/longitude coordinates and spatial predicates.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:geo_spatial_index","type":"geo_spatial_index","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`ADD`, `REMOVE`, `RADIUS`, `BOX`, `POLYGON`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:geo_spatial_index` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:geo_spatial_index",
  "type": "geo_spatial_index",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD`

Accepts `id`, `point:{lat,lon}`, optional string `metadata` map and returns `true`; replacing an ID moves its grid cell.

```json
{
  "key": "example:geo_spatial_index",
  "operation": "ADD",
  "args": {
    "id": "manila",
    "point": {
      "Lat": 14.5995,
      "Lon": 120.9842
    },
    "metadata": {
      "country": "PH"
    }
  }
}
```

#### `OP RADIUS`

Accepts `center`, `radius_meters`, optional `limit:100` and returns distance-sorted hits with `distance_meters`.

```json
{
  "key": "example:geo_spatial_index",
  "operation": "RADIUS",
  "args": {
    "center": {
      "Lat": 14.5995,
      "Lon": 120.9842
    },
    "radius_meters": 1000,
    "limit": 10
  }
}
```

#### `OP REMOVE`

Accepts `id` and returns whether the item existed.

```json
{
  "key": "example:geo_spatial_index",
  "operation": "REMOVE",
  "args": {
    "id": "manila"
  }
}
```

#### `OP BOX`

Accepts a bounding box and returns ID-sorted items inside inclusive bounds.

```json
{
  "key": "example:geo_spatial_index",
  "operation": "BOX",
  "args": {
    "box": {
      "MinLat": 14.0,
      "MinLon": 120.0,
      "MaxLat": 15.0,
      "MaxLon": 122.0
    }
  }
}
```

#### `OP POLYGON`

Accepts polygon points and returns ID-sorted items inside the polygon.

```json
{
  "key": "example:geo_spatial_index",
  "operation": "POLYGON",
  "args": {
    "points": [
      {
        "Lat": 14.5,
        "Lon": 120.9
      },
      {
        "Lat": 14.7,
        "Lon": 120.9
      },
      {
        "Lat": 14.6,
        "Lon": 121.1
      }
    ]
  }
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:geo_spatial_index"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:geo_spatial_index"}` — returns the wire type.
- `EXISTS`: `{"key":"example:geo_spatial_index"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:geo_spatial_index","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:geo_spatial_index"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:geo_spatial_index"}` — removes the TTL.
- `DEL`: `{"keys":["example:geo_spatial_index"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
