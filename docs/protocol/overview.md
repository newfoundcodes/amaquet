---
title: "Amaquet protocol overview"
---

Amaquet is Amaquet's binary framed protocol. It is not RESP and it does not proxy Redis.

Two URI schemes are recognized:

- `amaquet://` for plain TCP.
- `amaquets://` for TLS.

The default port is `13378`.

## Request flow

Connections negotiate the single supported protocol version before normal command traffic. Authentication follows negotiation when enabled, after which request IDs let command responses complete independently.

1. Open TCP/TLS connection.
2. Send `HELLO` to negotiate protocol version and inspect server metadata/identity (the Go client does this automatically).
3. If authentication is required, send `AUTH`.
4. Send command frames. Request IDs allow responses to be matched to requests.
5. For Pub/Sub, `SUBSCRIBE` creates an event stream whose event frames reuse the subscription request ID.

## Payload encoding

Frame payloads are JSON. Framing is binary; command values are typed JSON structures.

Protocol version 1 capabilities advertised by `HELLO` are `cancel`, `batch`, `cas`, `scan`, `chunked_blob`, and `pubsub`. The protocol supports multiplexed command responses but makes no cross-command transaction guarantee.

## Hard limits

The protocol implementation has a 64 MiB hard payload ceiling. The server can configure a smaller per-connection maximum with `limits.max_payload_bytes`.

The configured read timeout applies while waiting for the next frame/header/payload; successful reads refresh it. The write timeout applies to responses and Pub/Sub event frames. `limits.command_timeout_ms` creates a context deadline for each command and synchronous non-command opcode handler.
