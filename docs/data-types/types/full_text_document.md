---
title: "full_text_document"
---

**Category:** Indexes and search  
**Wire type:** `full_text_document`  
**Construction:** `CREATE`

Index/search types provide exact, range, prefix, full-text, vector, and secondary lookup paths.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:full_text_document","type":"full_text_document","options":{"id":"doc-1","metadata":{"source":"import"}}}'
```

## Construction options

Example `CREATE` options: `{"id":"doc-1","metadata":{"source":"import"}}`. Omitted fields use server defaults.

## Supported operations

`ADD_TEXT`, `ADD_TAG`, `ADD_NUMERIC`, `GET`, `SNAPSHOT`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:full_text_document` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:full_text_document",
  "type": "full_text_document",
  "options": {
    "id": "doc-1",
    "metadata": {
      "source": "import"
    }
  }
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP ADD_TEXT`

Accepts `name`, string `value`, optional `weight:1` and returns `true`; appends a field.

```json
{
  "key": "example:full_text_document",
  "operation": "ADD_TEXT",
  "args": {
    "name": "body",
    "value": "hello amaquet world",
    "weight": 2
  }
}
```

#### `OP ADD_TAG`

Accepts `name`, `values` string array and returns `true`; appends a field.

```json
{
  "key": "example:full_text_document",
  "operation": "ADD_TAG",
  "args": {
    "name": "tags",
    "values": [
      "go",
      "memory"
    ]
  }
}
```

#### `OP ADD_NUMERIC`

Accepts `name`, numeric `value` and returns `true`; appends a field.

```json
{
  "key": "example:full_text_document",
  "operation": "ADD_NUMERIC",
  "args": {
    "name": "score",
    "value": 9.5
  }
}
```

#### `OP SNAPSHOT`

Takes no arguments and returns complete document snapshot.

```json
{
  "key": "example:full_text_document",
  "operation": "SNAPSHOT",
  "args": {}
}
```

#### `OP GET`

Takes no arguments and returns the complete document snapshot.

```json
{
  "key": "example:full_text_document",
  "operation": "GET",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:full_text_document"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:full_text_document"}` — returns the wire type.
- `EXISTS`: `{"key":"example:full_text_document"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:full_text_document","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:full_text_document"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:full_text_document"}` — removes the TTL.
- `DEL`: `{"keys":["example:full_text_document"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
