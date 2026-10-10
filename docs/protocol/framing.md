---
title: "Binary framing"
---

Every Amaquet frame has a fixed 24-byte header followed by a bounded payload.

| Offset | Size | Field            | Encoding                             |
| -----: | ---: | ---------------- | ------------------------------------ |
|      0 |    4 | magic            | ASCII `AMQT`                         |
|      4 |    1 | protocol version | currently `1`                        |
|      5 |    1 | kind             | request `1`, response `2`, event `3` |
|      6 |    2 | flags            | unsigned big-endian                  |
|      8 |    2 | opcode           | unsigned big-endian                  |
|     10 |    2 | reserved         | currently zero                       |
|     12 |    8 | request ID       | unsigned big-endian                  |
|     20 |    4 | payload length   | unsigned big-endian                  |

## Opcodes

The opcode identifies the operation carried by a frame and determines how the server handles its payload.

| Opcode | Name          | Purpose                                   |
| -----: | ------------- | ----------------------------------------- |
|      1 | `COMMAND`     | Run an Amaquet command                    |
|      2 | `HELLO`       | Server metadata / nonce signature         |
|      3 | `AUTH`        | Authenticate an API key                   |
|      4 | `SUBSCRIBE`   | Subscribe to Pub/Sub                      |
|      5 | `UNSUBSCRIBE` | Remove subscription                       |
|      6 | `CANCEL`      | Cancel an in-flight command by request ID |

Requests carry independent request IDs. The server can execute multiple command requests concurrently on one connection and serialize frames safely on the socket. Responses can therefore complete in a different order from requests; clients must correlate by request ID.

Only `COMMAND` requests execute concurrently. Other opcodes are handled synchronously in connection read order. Reusing a request ID while its command is still active returns `DUPLICATE_REQUEST`. Per-connection/global active-request semaphores return a framed overload error; if the global declared-payload reservation cannot be acquired, the reader stops and the connection closes before the payload is admitted.

A `CANCEL` payload is `{"request_id":42}`; its own frame has a distinct request ID. The result reports the target ID and whether an active request was found. Cancellation is cooperative: the request context is cancelled, so an operation must observe that context to stop early.

Invalid magic, unknown versions/opcodes, malformed request frames, truncated input, and payloads above the configured/hard limit are rejected. Read deadlines prevent a peer from holding a connection indefinitely with a partial frame.

## Response payload

Command responses use a common success-or-error JSON envelope inside the binary frame payload.

```json
{ "ok": true, "result": {} }
```

Errors use:

```json
{ "ok": false, "error": { "code": "NOT_FOUND", "message": "key not found" } }
```

## Pub/Sub frames

`SUBSCRIBE` sends `{"key":"bus","channel":"updates","buffer":16}`. The response confirms the key/channel, then event frames use kind `3`, opcode `SUBSCRIBE`, and the original subscription request ID. An event payload contains `channel`, typed `payload`, and `published_at`. Re-subscribing the same connection to the same key/channel replaces its old server subscription.

`UNSUBSCRIBE` sends `{"key":"bus","channel":"updates"}` and returns `{"unsubscribed":true|false}`. Connection close removes every subscription. Server and Go-client channels are bounded and non-blocking, so slow consumers can lose events; `pubsub` is deliberately best effort.
