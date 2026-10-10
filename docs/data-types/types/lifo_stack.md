---
title: "lifo_stack"
---

**Category:** Queues and messaging  
**Wire type:** `lifo_stack`  
**Construction:** `CREATE`

Queue and messaging types provide ordering, delivery, streaming, and publication semantics.

## Create a value

Use the top-level `CREATE` command to initialize this structured value. The example shows its construction options.

```bash
amaquet-cli -uri amaquet://127.0.0.1:13378 CREATE '{"key":"example:lifo_stack","type":"lifo_stack","options":{}}'
```

## Construction options

Example `CREATE` options: `{}`. Omitted fields use server defaults.

## Supported operations

`PUSH`, `POP`, `PEEK`, `LEN`.

See the [complete operation contract](../operations.md) for exact arguments, defaults, results, and whether a listed `GET` is the top-level key command or a type-specific `OP`. It also identifies read-only operations and documents validation, lazy maintenance, and key-version behavior.

## Operation payloads

The following examples use the public `amaquet-cli` JSON command format. Replace `example:lifo_stack` with the key used by your application.

### Construction payloads

Use the payload below to initialize this type before issuing reads or type-specific operations.

#### `CREATE`

This payload initializes the value and must complete before its type-specific operations are used.

```json
{
  "key": "example:lifo_stack",
  "type": "lifo_stack",
  "options": {}
}
```

### Operations

Each request uses the `OP` envelope. The response is the type-specific value described in the [complete operation contract](../operations.md).

#### `OP PUSH`

Accepts nested `value` and returns new length.

```json
{
  "key": "example:lifo_stack",
  "operation": "PUSH",
  "args": {
    "value": {
      "type": "utf8_string",
      "value": "first"
    }
  }
}
```

#### `OP PEEK`

Takes no arguments and returns newest typed value or `null`.

```json
{
  "key": "example:lifo_stack",
  "operation": "PEEK",
  "args": {}
}
```

#### `OP POP`

Takes no arguments and returns newest typed value or `null`.

```json
{
  "key": "example:lifo_stack",
  "operation": "POP",
  "args": {}
}
```

#### `OP LEN`

Takes no arguments and returns the stack length.

```json
{
  "key": "example:lifo_stack",
  "operation": "LEN",
  "args": {}
}
```

### Top-level read payload

`GET` reads the current value without changing it:

```json
{
  "key": "example:lifo_stack"
}
```

### Key lifecycle payloads

These commands apply to the key regardless of its registered type:

- `TYPE`: `{"key":"example:lifo_stack"}` — returns the wire type.
- `EXISTS`: `{"key":"example:lifo_stack"}` — reports whether the key exists.
- `EXPIRE`: `{"key":"example:lifo_stack","ttl_ms":5000}` — assigns a five-second TTL.
- `TTL`: `{"key":"example:lifo_stack"}` — returns the remaining TTL in milliseconds.
- `PERSIST`: `{"key":"example:lifo_stack"}` — removes the TTL.
- `DEL`: `{"keys":["example:lifo_stack"]}` — deletes the key.

For defaults, validation, result shapes, and persistence classification, see the [complete operation contract](../operations.md).

## Key lifecycle

The key participates in `TYPE`, `EXISTS`, `TTL`, `EXPIRE`, `PERSIST`, and `DEL`. Top-level `GET` returns the type's direct public value or a synchronized snapshot.
