---
title: "Amaquet authentication and identity"
---

This page describes protocol negotiation, API-key authentication, permissions, and optional Ed25519 identity verification.

## `HELLO`

`HELLO` does not require authentication. Send:

```json
{ "min_protocol": 1, "max_protocol": 1, "nonce": "client-generated-random-value" }
```

Omitted protocol bounds default to version 1. The response reports `name`, server `version`, frame `protocol`, `selected_protocol`, capability names, `auth_required`, and optional Ed25519 identity fields. There is currently only protocol version 1; non-overlapping bounds return an error.

## `AUTH`

Payload:

```json
{ "api_key": "amaquet_..." }
```

Despite the field name, the credential may be a bootstrap token, API key, or an active member token that does not have MFA enabled. Member tokens with MFA enabled must use the HTTP session endpoint because native Amaquet `AUTH` has no MFA-code field. API-key secrets are generated with 32 random bytes and only their SHA-256 hashes are persisted; the raw secret is returned once when the key is created. On success, the connection stores the current role and actor ID. Later commands revalidate the actor and authorize it against the current RBAC table, so revocation, expiration, disabling, and role changes take effect without reconnecting.

When `protocol.require_auth` is false, a new connection is an anonymous `admin` actor and can issue commands without `AUTH`. This mode should be confined to trusted environments.

## Read/write classification

`PING`, `GET`, `EXISTS`, `TYPE`, `TTL`, `KEYS`, `SCAN`, `INFO`, `TYPES`, `MEMORY`, `BLOB_READ`, and the fixed read-only `OP` set require `data.read`. All other commands/operations require `data.write`.

`SUBSCRIBE` requires `data.read`. `UNSUBSCRIBE` closes an already established local subscription without a separate authorization check. Authentication failures are throttled per remote IP using `admin.auth_failures_per_minute`, which is shared with the administration server setting. The native protocol has no MFA challenge-response flow; MFA-protected members must authenticate through `/api/session` and use the HTTP session cookie for administrative requests.

## Client identity verification

With identity keys configured, verify the returned nonce signature before trusting the application-level server identity. For network confidentiality and certificate-based server verification, also use TLS through `amaquets://`.
