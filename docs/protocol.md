---
title: "Amaquet protocol specification v1"
---

This specification defines Amaquet v1 connection URIs, binary frames, authentication, commands, and event delivery.

## URI schemes

Use these URI forms to select plaintext TCP or TLS when connecting a client to the native protocol listener.

- `amaquet://host[:port]` — Amaquet over TCP.
- `amaquets://host[:port]` — Amaquet over TLS.
- Default port: `13378`.
- API keys should be passed out of band through `DialOptions.APIKey` or the CLI's `-api-key` flag. URI credentials exist only as explicit opt-in legacy compatibility.

Examples:

```text
amaquet://127.0.0.1:13378
amaquet://example.com
amaquets://db.example.com:13378
```

## Frame format

Every frame starts with a fixed 24-byte header. Multi-byte integers are big-endian.

| Offset | Size | Field                  |
| -----: | ---: | ---------------------- |
|      0 |    4 | ASCII magic `AMQT`     |
|      4 |    1 | protocol version (`1`) |
|      5 |    1 | frame kind             |
|      6 |    2 | flags                  |
|      8 |    2 | opcode                 |
|     10 |    2 | reserved               |
|     12 |    8 | request ID             |
|     20 |    4 | payload length         |

The payload follows immediately. The current hard frame payload limit is 64 MiB.

Frame kinds:

- `1`: request
- `2`: response
- `3`: asynchronous event

Opcodes:

- `1`: command
- `2`: hello
- `3`: authentication
- `4`: subscribe
- `5`: unsubscribe
- `6`: cancel an active command request

The protocol is not RESP and has no Redis dependency.

## Multiplexing

The 64-bit request ID allows multiple in-flight requests on one connection. Responses carry the same request ID. Pub/Sub events use the subscription request ID and frame kind `event`.

## HELLO

Request payload (the negotiation fields default to protocol 1 when omitted):

```json
{ "nonce": "client-generated-nonce", "min_protocol": 1, "max_protocol": 1 }
```

Response fields include server/version/protocol/auth requirements. If an Ed25519 identity is configured, it also returns:

```json
{
  "identity_public_key": "base64-raw-public-key",
  "identity_fingerprint": "ed25519:...",
  "nonce_signature": "base64-raw-signature"
}
```

The Go client exposes `ServerInfo.VerifyNonce`.

## AUTH

Authentication request:

```json
{ "api_key": "amaquet_..." }
```

Response:

```json
{ "authenticated": true, "role": "developer" }
```

If authentication is required, normal commands and subscriptions are rejected until AUTH succeeds.

## COMMAND

Command payload:

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

Success response envelope:

```json
{ "ok": true, "result": true }
```

Error response envelope:

```json
{
  "ok": false,
  "error": {
    "code": "WRONG_TYPE",
    "message": "wrong value type"
  }
}
```

Known error classes include `NOT_FOUND`, `WRONG_TYPE`, `EXISTS`, `CONFLICT`, `MEMORY_LIMIT`, `BUSY`, `DUPLICATE_REQUEST`, `FORBIDDEN`, `UNAUTHORIZED`, and `ERROR`.

## Direct values vs composite values

Scalar/directly serialized values use `SET`. Examples include strings, integers, decimals, timestamps, UUIDs, JSON, geospatial value objects, standalone vectors, CIDRs, and URLs.

Composite data structures use `CREATE` followed by `OP`.

```json
{
  "command": "CREATE",
  "args": {
    "key": "users",
    "type": "set",
    "options": {}
  }
}
```

Then:

```json
{
  "command": "OP",
  "args": {
    "key": "users",
    "operation": "ADD",
    "args": {
      "value": { "type": "utf8_string", "value": "alice" }
    }
  }
}
```

## Common keyspace commands

`PING`, `SET`, `CAS`, `BATCH`, `CREATE`, `GET`, `DEL`, `EXISTS`, `TYPE`, `TTL`, `EXPIRE`, `PERSIST`, `KEYS`, `SCAN`, `BLOB_BEGIN`, `BLOB_CHUNK`, `BLOB_COMMIT`, `BLOB_READ`, `BLOB_ABORT`, `TYPES`, `INFO`, `MEMORY`, and `OP` are implemented by the v1 dispatcher.

`GET` returns a safe snapshot instead of exposing internal mutable pointers.

## SUBSCRIBE / UNSUBSCRIBE

Subscription request:

```json
{ "key": "bus", "channel": "updates", "buffer": 64 }
```

Events are Amaquet event frames:

```json
{
  "channel": "updates",
  "payload": { "type": "utf8_string", "value": "changed" },
  "published_at": "2026-10-01T00:00:00Z"
}
```

## RBAC

The protocol server maps keyspace reads to `data.read` and mutations to `data.write`. `OP` operations use their read/mutation classification. The current role permission sets are controlled by the admin state and can be changed through the administration API.
