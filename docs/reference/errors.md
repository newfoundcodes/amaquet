---
title: "Error model"
---

Amaquet response errors include a stable code and human-readable message.

Common codes:

| Code                | Meaning                                                                                  |
| ------------------- | ---------------------------------------------------------------------------------------- |
| `NOT_FOUND`         | key/resource was not found                                                               |
| `WRONG_TYPE`        | operation was incompatible with stored type                                              |
| `EXISTS`            | conditional create/write found an existing key                                           |
| `CONFLICT`          | version conflict                                                                         |
| `MEMORY_LIMIT`      | a write would exceed `max_memory_bytes` and eviction could not make room                 |
| `BUSY`              | an in-flight request, authentication-rate, or other framed server-busy limit was reached |
| `DUPLICATE_REQUEST` | a multiplexed connection reused an active request ID                                     |
| `FORBIDDEN`         | authenticated role lacks permission                                                      |
| `UNAUTHORIZED`      | authentication is required/invalid                                                       |
| `ERROR`             | validation, decoding, operation, or other server error                                   |

These codes are produced by native Amaquet response frames. Unknown commands, malformed arguments, timeouts/cancellation, and most data-type validation failures currently use the general `ERROR` code, so clients may need to surface the diagnostic message for those failures.

The global inbound-payload budget is enforced while reading frames. If that reservation cannot be acquired, the server closes the affected connection instead of decoding the payload and returning a `BUSY` frame.

The administration HTTP API uses the same `{ok:false,error:{code,message}}` shape but has a route-specific vocabulary such as `BAD_REQUEST`, `INVALID`, `METHOD_NOT_ALLOWED`, `RATE_LIMITED`, `SESSION_LIMIT`, `INVALID_INVITE`, `MFA_ERROR`, `SAVE_FAILED`, `DISABLED`, `CHECKPOINT_FAILED`, `COMPACT_FAILED`, and `COMMAND_ERROR`. Use the HTTP status first, then the structured code; do not parse the message.
