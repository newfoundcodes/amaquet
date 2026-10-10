---
title: "Administration API"
description: "Operate Amaquet through its HTTP administration API for authentication, organizations, members, API keys, RBAC, data access, metrics, and audit events."
---

HTTP clients can authenticate each request with `Authorization: Bearer <token>` or `X-Amaquet-Admin-Token`. A client that needs MFA or cookie-based authentication can exchange its credential through the server-side session endpoint.

Most structured request bodies use the shared decoder, which caps input at 2 MiB and rejects unknown fields. The data-command proxy instead accepts up to 8 MiB with ordinary JSON decoding, and the member-invitation endpoint uses an uncapped ordinary decoder for its optional `ttl_hours` object. JSON-producing routes set `Content-Type: application/json`; errors normally have the common shape:

```json
{ "ok": false, "error": { "code": "BAD_REQUEST", "message": "diagnostic text" } }
```

The administration listener serves its embedded OpenAPI 3.1 document at `GET /openapi.json`; `HEAD` returns the same headers with no body. This discovery endpoint is public and does not require authentication.

Bearer authentication accepts bootstrap, API-key, and member credentials. A member with MFA enabled cannot use its token as a direct bearer credential; it must create an HTTP cookie session with the token plus a TOTP code. Failed authentication is limited per `RemoteAddr` IP by `admin.auth_failures_per_minute`.

The methods listed below are the supported API contract and the methods published in OpenAPI. Several read-only handlers currently dispatch by path without an explicit method check, so an alternate verb can reach the same handler; clients must not depend on that implementation detail.

## Authentication and identity

These endpoints establish, inspect, and manage administrator and member authentication state.

| Method | Path                          | Purpose                                                               |
| ------ | ----------------------------- | --------------------------------------------------------------------- |
| POST   | `/api/session`                | exchange bootstrap/API/member credential for an HTTP cookie session   |
| DELETE | `/api/session`                | destroy the current HTTP cookie session                               |
| GET    | `/api/auth/check`             | verify the current session/token                                      |
| POST   | `/api/identity/accept-invite` | accept a one-time member invitation and receive a member access token |
| POST   | `/api/identity/mfa`           | start TOTP MFA setup for the current member                           |
| PUT    | `/api/identity/mfa`           | confirm TOTP MFA with a six-digit code                                |
| DELETE | `/api/identity/mfa`           | disable TOTP MFA for the current member                               |

`POST /api/session` accepts:

```json
{ "token": "amaquet_...", "mfa_code": "123456" }
```

`mfa_code` is required only for member identities with TOTP enabled.

The response includes `ok`, `role`, and `expires_at` and sets `amaquet_session`. Sessions last 12 hours, are server-side, and are limited by `admin.max_sessions`. The cookie is `HttpOnly`, `SameSite=Strict`, scoped to `/`, and `Secure` on HTTPS. Revoked API-key actors, disabled/deleted members, rotated member credentials, and MFA changes invalidate matching sessions through the revocation path.

Invite acceptance accepts `{"invite":"amaquet_invite_…"}` and returns the member plus a one-time `access_token`. Starting MFA returns a base32 `secret` and an `otpauth` URI; confirmation accepts `{"code":"123456"}`. The MFA route requires `org.read` at the wrapper and then rejects any actor that is not a member.

## Organization and members

These endpoints manage the organization record, members, invitations, and role-permission assignments.

| Method | Path                       | Permission      |
| ------ | -------------------------- | --------------- |
| GET    | `/api/organization`        | `org.read`      |
| PUT    | `/api/organization`        | `org.write`     |
| GET    | `/api/members`             | `members.read`  |
| POST   | `/api/members`             | `members.write` |
| PATCH  | `/api/members/{id}`        | `members.write` |
| DELETE | `/api/members/{id}`        | `members.write` |
| POST   | `/api/members/{id}/invite` | `members.write` |
| GET    | `/api/rbac`                | `rbac.read`     |
| PUT    | `/api/rbac`                | `rbac.write`    |

An invitation response contains the invitation token only once. `ttl_hours` defaults to 24.

Organization updates accept `{"name":"Acme","slug":"acme"}`. Member creation accepts `name`, `email`, and `role`. A member patch must include a valid `role` on every request and may include `status`; roles are `admin`, `auditor`, or `developer`. The server stores a non-empty status as supplied, but only the exact status `active` can authenticate; API clients should use `active` and `disabled`. The RBAC update body is the complete role-to-permissions map, for example `{"developer":["data.read","data.write"]}`, and all three role keys are required.

## API keys

These endpoints create, list, and revoke API-key credentials for protocol and administration access.

| Method | Path                 | Permission   |
| ------ | -------------------- | ------------ |
| GET    | `/api/api-keys`      | `keys.read`  |
| POST   | `/api/api-keys`      | `keys.write` |
| DELETE | `/api/api-keys/{id}` | `keys.write` |

A created API-key secret is returned once. Revocation invalidates new authentication, existing HTTP cookie sessions, and matching active Amaquet connections.

Creation accepts:

```json
{ "name": "CI developer", "role": "developer", "expires_at": "2027-01-01T00:00:00Z" }
```

`expires_at` may be omitted or `null`. The response is `{"api_key":{…},"secret":"amaquet_…","warning":"…"}`. List results omit stored credential hashes.

## System and persistence

These endpoints expose the running configuration and start server-side persistence maintenance actions.

| Method | Path                          | Permission     |
| ------ | ----------------------------- | -------------- |
| GET    | `/api/system`                 | `system.read`  |
| PUT    | `/api/system`                 | `system.write` |
| POST   | `/api/persistence/checkpoint` | `system.write` |
| POST   | `/api/persistence/compact`    | `system.write` |

`GET /api/system` returns `{"config":{…},"restart_required":false}`. `PUT` accepts the complete configuration object, validates and crash-safely saves it, clears `security.bootstrap_token`, and returns the public config with `restart_required`. The flag is true whenever the public submitted config differs from the current config; the endpoint persists changes but does not rebind listeners or rebuild engine components in place.

Checkpoint creation produces a compact, verified AMQTAOF2 recovery image. Offline restore uses `amaquet-restore`; live in-place restore is intentionally not offered through HTTP.

The checkpoint request optionally accepts `{"name":"nightly.aof"}`. Only the base filename is used; an omitted name becomes a UTC timestamped filename under `data_dir/backups`. The response contains its server-local `path`. Compaction has no request body and returns `{"ok":true}`. Both return HTTP 409 with `DISABLED` when AOF is off.

## Data operations

These endpoints provide an HTTP control-plane proxy for key inspection, deletion, and native command execution.

| Method | Path                | Permission                       |
| ------ | ------------------- | -------------------------------- |
| GET    | `/api/data/keys`    | `data.read`                      |
| GET    | `/api/data/key`     | `data.read`                      |
| DELETE | `/api/data/key`     | `data.write`                     |
| POST   | `/api/data/command` | dynamic `data.read`/`data.write` |

`GET /api/data/keys` accepts `cursor`, `prefix`, and `count`; an omitted or unparsable value stays at the handler default of 250, a non-positive value selects the engine default of 100, and values above 10,000 are capped at 10,000. It returns `{"keys":[…],"cursor":"…"}`. `GET`/`DELETE /api/data/key` require a URL-encoded `key` query parameter.

The command body is the native `{"command":"GET","args":{…}}` request and is capped at 8 MiB before ordinary JSON decoding. Top-level read commands and a fixed list of read-only `OP` names use `data.read`; everything else uses `data.write`. The current HTTP classifier does not list `BLOB_READ`, so that command requires `data.write` through this route even though native Amaquet authorization treats it as `data.read`. Success wraps the native result as `{"ok":true,"result":…}`.

## Health and metrics

These read-only endpoints expose liveness, readiness, operational summaries, and Prometheus metrics.

- `GET /api/health` returns process liveness, service name, and UTC time without authentication.
- `GET /api/ready` returns key and memory statistics; it returns HTTP 503 when AOF persistence health is degraded.
- `GET /api/overview` requires `data.read` and returns counts plus organization, memory, and compression summaries.
- `GET /metrics` emits Prometheus 0.0.4 text. When `admin.metrics_require_auth` is true, the caller needs `system.read`.

Metrics are `amaquet_keys`, `amaquet_memory_bytes`, `amaquet_memory_limit_bytes`, `amaquet_evicted_keys_total`, `amaquet_compressed_keys`, `amaquet_compression_saved_bytes`, `amaquet_go_heap_bytes`, `amaquet_go_goroutines`, and `amaquet_ready`.

## Audit

`GET /api/audit?limit=200` requires `audit.read`. A positive limit returns at most that many newest-first retained events; a non-positive limit or a value above the retained count returns all retained events. Organization, member, invitation, MFA, RBAC, and API-key mutations create audit entries. Configuration updates and persistence operations currently do not append an audit event.

## Security headers

The administration listener serves API and metrics routes only. Unregistered paths, including `/`, return 404.

All responses receive `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, and a same-origin Content Security Policy. Cookie-authenticated requests rely on `SameSite=Strict` as their built-in cross-site-request mitigation; the server does not validate `Origin`/`Referer` and does not issue a separate CSRF token. Bearer credentials are not sent automatically by a browser, but clients are still responsible for keeping them out of untrusted content.
