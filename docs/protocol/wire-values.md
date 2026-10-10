---
title: "Wire values and snapshots"
---

Amaquet frames carry JSON payloads. A top-level `SET` identifies its value with sibling `type` and `value` arguments, while values nested inside collections use an explicit envelope:

```json
{ "type": "utf8_string", "value": "hello" }
```

The `type` string must be one of the names returned by `TYPES`. Unknown names and malformed representations are rejected before mutation.

## Direct `SET` representations

This table defines the JSON representation and validation rules for values stored directly with `SET`.

| Type                     | JSON representation                       | Validation/notes                                                          |
| ------------------------ | ----------------------------------------- | ------------------------------------------------------------------------- |
| `binary_string`          | base64 string                             | Standard padded base64                                                    |
| `utf8_string`            | string                                    | Must contain valid UTF-8                                                  |
| `integer`                | JSON integer                              | Signed 64-bit                                                             |
| `unsigned_integer`       | JSON integer                              | Unsigned 64-bit                                                           |
| `float`                  | JSON number                               | Go `float64`                                                              |
| `decimal`, `big_decimal` | decimal string                            | Exact coefficient and scale; no exponent notation                         |
| `boolean`                | boolean                                   | `true` or `false`                                                         |
| `null`                   | `null`                                    | Explicit null value                                                       |
| `timestamp`              | string                                    | RFC 3339 with optional fractional nanoseconds                             |
| `duration`               | string                                    | Go duration syntax such as `250ms` or `2h45m`                             |
| `uuid`                   | string                                    | Canonical/hyphenated or 32 hexadecimal digits                             |
| `big_integer`            | decimal string                            | Arbitrary-precision base-10 integer                                       |
| `symbol`                 | string                                    | Non-empty and contains no whitespace                                      |
| `json`                   | any JSON value                            | Numbers are decoded with their textual precision preserved                |
| `messagepack`, `cbor`    | base64 string                             | Stored as opaque bytes; Amaquet does not validate the external encoding   |
| `geo_point`              | `{"lat":n,"lon":n}`                       | Latitude −90…90, longitude −180…180                                       |
| `bounding_box`           | `{minLat,minLon,maxLat,maxLon}`           | Input matching is case-insensitive; bounds must be ordered and geographic |
| `polygon`                | array of points                           | At least three points                                                     |
| `dense_vector`           | number array                              | Float32 elements                                                          |
| `sparse_vector`          | `{"dim":n,"values":{"0":v}}`              | Positive dimension; indexes in range; zero entries omitted                |
| `quantized_vector`       | `{"scale":n,"zero_point":n,"data":[...]}` | Signed 8-bit data elements                                                |
| `cidr_network`           | string                                    | Parsed and canonicalized by Go's CIDR parser                              |
| `url`                    | string                                    | Request URI with a required scheme                                        |

`json`, `messagepack`, and `cbor` also support `CREATE`, which initializes `{}` or empty opaque bytes before later type-specific operations. Other registered types are created with `CREATE` because they need internal structure or construction options.

## Command request and response

The following examples show the JSON command envelope, successful result, and snapshot returned by the protocol.

```json
{
  "command": "SET",
  "args": {
    "key": "answer",
    "type": "integer",
    "value": 42
  }
}
```

The protocol wraps the command result:

```json
{ "ok": true, "result": true }
```

`GET` returns:

```json
{
  "key": "answer",
  "type": "integer",
  "value": 42,
  "version": 1,
  "expires_at": "2026-10-04T12:00:00Z"
}
```

`expires_at` is omitted for persistent keys. The value version begins at 1 and increases for successful replacement, TTL changes, persistence changes, and successful mutating `OP` calls. Pure reads do not increase it.

Because `BoundingBox` currently has no explicit JSON field tags, its `GET` value uses Go's exported field names: `MinLat`, `MinLon`, `MaxLat`, and `MaxLon`. The lower-camel input form in the table is accepted case-insensitively.

## Snapshot behavior

`GET` never exposes internal Go pointers. Mutable values return a synchronized snapshot. Important bounded snapshots include:

- streams and event logs return at most their first 1,000 entries through ordinary `GET`;
- large chunked binary strings above 32 MiB return `{chunked,size,read_command}` metadata and must be read with `BLOB_READ`;
- probabilistic structures return configuration/summary data rather than their complete backing arrays;
- active Pub/Sub values return channel-to-subscriber counts, not buffered messages;
- HNSW, B-tree, hash, and inverted indexes return summary metadata; radix trees cap their snapshot at 1,000 prefix results. Geo indexes, exact/flat vector sets, secondary indexes, and adjacency sets currently return complete logical snapshots, so ordinary `GET` can be large for those types.

Use the type-specific read operation when it provides more appropriate pagination/range semantics.

## Nested identity and equality

Collections store a complete `core.Value`, not an untyped JSON scalar. Sets and map-like structures derive internal equality keys from the stable type ID plus a deterministic representation of the value. Values with different Amaquet types are distinct even if their JSON rendering looks similar; for example, `integer:1` and `unsigned_integer:1` are different members.

## JSON number preservation

JSON documents use `json.Decoder.UseNumber` when decoded and copied. This prevents an integer-looking JSON token from being eagerly converted to a binary float merely by document storage. JSON remains a document value; it is not recursively converted into Amaquet typed values.
