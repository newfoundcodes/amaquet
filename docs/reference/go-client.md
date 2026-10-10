---
title: "Go client library"
description: "Integrate Amaquet from Go with the concurrent-safe client, TCP or TLS dialing, request multiplexing, authentication, subscriptions, and typed commands."
---

The public package `github.com/newfoundcodes/amaquet/pkg/amaquet` is the supported Go client. It owns a single TCP/TLS connection, negotiates protocol v1 with `HELLO`, multiplexes requests by 64-bit request ID, routes asynchronous event frames, and is safe for concurrent command calls. `Close` is idempotent.

## Connect

Create a client with a bounded context, an explicit API key, and a TLS configuration when using `amaquets://`.

```go
import (
    "context"
    "crypto/tls"
    "time"

    "github.com/newfoundcodes/amaquet/pkg/amaquet"
)

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

client, err := amaquet.DialWithOptions(ctx, "amaquets://db.example.com:13378", amaquet.DialOptions{
    APIKey: "amaquet_...",
    Timeout: 30 * time.Second,
    TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
})
if err != nil {
    // handle connection, TLS, HELLO, or AUTH failure
}
defer client.Close()
```

`Dial` is shorthand for `DialWithOptions` with defaults. URI-embedded credentials are rejected by default because URLs commonly leak through logs, shell history, and telemetry. Prefer `DialOptions.APIKey`. Set `AllowURISecrets` only for explicit legacy compatibility.

For `amaquets://`, the client derives `ServerName` from the URI host when it is absent and enforces TLS 1.2 or newer when no higher minimum is supplied. Normal certificate validation remains enabled unless the caller deliberately changes `TLSConfig`.

## Dial options

Configure authentication, TLS, timeouts, and the legacy URI-secret opt-in through `DialOptions`.

| Field             | Meaning                                                                    |
| ----------------- | -------------------------------------------------------------------------- |
| `APIKey`          | Credential sent by `AUTH` after `HELLO`                                    |
| `TLSConfig`       | Optional cloned TLS configuration for `amaquets://`                        |
| `Timeout`         | Client-side response deadline; defaults to 30 seconds                      |
| `AllowURISecrets` | Permit credentials parsed from URI user-info or `api_key`; default `false` |

The dial context bounds TCP connection, TLS handshake, and initial negotiation. Each later call also accepts its own context. On context cancellation or client timeout, ordinary request calls send a best-effort Amaquet `CANCEL` frame for that request ID. `Subscribe` waits for its synchronous acknowledgement using both the supplied context and `Client.Timeout`; callers should still give that context a deadline.

## Public API

The convenience methods below cover connection setup, common commands, Pub/Sub, and chunked binary transfer. `Command` remains the escape hatch for every protocol command that does not have a dedicated helper.

| Method                    | Behavior                                                                   |
| ------------------------- | -------------------------------------------------------------------------- |
| `Dial`, `DialWithOptions` | Connect, negotiate v1, optionally authenticate, and start the frame reader |
| `Close`                   | Close the connection and release pending requests/subscriptions            |
| `Command`                 | Send any `{command,args}` request and decode its result                    |
| `Hello`                   | Request server metadata and an optional nonce signature                    |
| `ServerInfo.VerifyNonce`  | Verify the Ed25519 signature returned for the supplied nonce               |
| `Ping`                    | Execute `PING`                                                             |
| `Set`                     | Store a directly encoded type with an optional TTL                         |
| `Get`                     | Return key/type/value/version metadata as `map[string]any`                 |
| `Delete`                  | Delete one or more keys                                                    |
| `Create`                  | Construct a composite/specialized type                                     |
| `Op`                      | Execute a type-specific operation                                          |
| `Subscribe`               | Open a Pub/Sub event stream backed by a buffered Go channel                |
| `Subscription.Close`      | Send `UNSUBSCRIBE` and close local event delivery                          |
| `BlobRead`                | Read at most one bounded binary range                                      |
| `UploadBlob`              | Stream a reader through begin/chunk/commit with abort-on-failure           |

## Typed commands

Convenience methods cover common operations; `Command` exposes the complete protocol:

```go
if err := client.Set(ctx, "counter", "integer", int64(41), 0); err != nil {
    // handle error
}

var value int64
err = client.Op(ctx, "counter", "ADD", map[string]any{"delta": 1}, &value)
```

Nested values use the [wire-value envelope](../protocol/wire-values.md):

```go
err = client.Create(ctx, "jobs", "fifo_queue", nil)
err = client.Op(ctx, "jobs", "ENQUEUE", map[string]any{
    "value": map[string]any{
        "type":  "json",
        "value": map[string]any{"id": "job-1"},
    },
}, nil)
```

`Command` returns protocol failures as Go errors formatted as `CODE: message`. The current client does not expose a separately typed error-code struct, so code that must branch on codes should wrap this boundary or use the lower-level protocol contract deliberately.

## Server identity

Verify the optional Ed25519 server identity by supplying an unpredictable application-generated nonce to `HELLO`.

```go
nonce := "application-generated-unpredictable-nonce"
info, err := client.Hello(ctx, nonce)
if err != nil || !info.VerifyNonce(nonce) {
    // reject an untrusted identity
}
```

Nonce verification proves possession of the configured Ed25519 identity key. It does not encrypt traffic or replace TLS hostname/certificate validation.

## Pub/Sub

Subscribe to a key and channel to receive asynchronous events through a bounded Go channel.

```go
sub, err := client.Subscribe(ctx, "bus", "updates", 64)
if err != nil {
    // handle error
}
defer sub.Close()

for message := range sub.C {
    // message.Channel, message.Payload, message.PublishedAt
}
```

The client uses non-blocking delivery to the configured local channel buffer. If the application does not consume quickly enough, client-side event delivery can be dropped. The server-side Pub/Sub object also uses bounded subscriber channels and reports its own `published`, `delivered`, and `dropped` counters.

## Large binary values

Use the blob helpers to stream large binary values without putting the full payload in a single command frame.

```go
err = client.UploadBlob(ctx, "archive", "", source, size, 1<<20)
chunk, total, err := client.BlobRead(ctx, "archive", 0, 1<<20)
```

`UploadBlob` chooses a `client-<request-counter>` upload ID when empty, defaults chunks to 1 MiB, caps chunks at 32 MiB, reads exactly the declared size, and attempts `BLOB_ABORT` if any stage fails. Its helper does not expose `nx`, `xx`, TTL, or `connection_scoped`; use `Command` directly when those `BLOB_BEGIN` options are required.

## Concurrency and shutdown

A background reader is the sole frame reader. A mutex serializes writes, while request IDs allow many callers to await independent responses. Responses may complete out of order. Closing the client closes all pending request and subscription channels; callers receive `net.ErrClosed` where applicable.

Avoid mutating a shared result object from multiple calls, and always bound long-running operations with a context even though the client and server both have default timeouts.
