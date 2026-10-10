---
title: "messagepack"
---

**Category:** Structured  
**Wire type:** `messagepack`  
**Construction:** `CREATE` (also supports top-level `SET`)

Structured values cover documents, encoded payloads, records, matrices, and tensors.

## Create a value

Use `CREATE` to initialize empty opaque bytes, or use top-level `SET` with standard base64 to store bytes directly.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:messagepack","type":"messagepack","options":{}}'
```

For direct initialization, supply standard base64 without asking Amaquet to validate the MessagePack encoding:

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 SET '{"key":"example:messagepack","type":"messagepack","value":"AQID"}'
```

## Construction options

`CREATE` accepts no type-specific options and starts with empty bytes. `SET` decodes the supplied standard-base64 string and stores the resulting opaque bytes.

## Supported operations

`GET`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:messagepack` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:messagepack",
  "type": "messagepack",
  "options": {}
}
```

### Operations

The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP GET`

Takes no arguments and returns the current opaque value.

```json
{
  "key": "example:messagepack",
  "operation": "GET",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:messagepack"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:messagepack"}` — returns the wire type.
- `EXISTS`: `{"key":"example:messagepack"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:messagepack","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:messagepack"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:messagepack"}` — removes the TTL.
- `DEL`: `{"keys":["example:messagepack"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
