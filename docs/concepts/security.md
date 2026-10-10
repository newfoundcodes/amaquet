---
title: "Security model"
---

Amaquet separates transport security, machine credentials, human member identities, HTTP cookie sessions, RBAC, and optional server identity.

## API keys

API keys are machine credentials. A key contains a random secret and a stable key ID. Amaquet stores the SHA-256 digest, role, expiration, last-use time, and revocation state.

Authentication hashes the presented token with SHA-256 and uses the digest as an index into API-key and member-credential maps, avoiding a scan of all credentials. Bootstrap-token digest comparison uses `subtle.ConstantTimeCompare`. The stored hashes are credential verifiers rather than encryption; the generated high-entropy tokens remain the security boundary.

Revoking a key prevents new authentication and invalidates the actor on existing sessions/connections. The server also closes matching active Amaquet connections through the revocation hook.

For remote protocol access, use `amaquets://`. Authenticated plaintext on a non-loopback listener is rejected unless `allow_insecure_auth` is explicitly enabled.

## Human member identities

A member record has a role and status. Administrators can create a one-time invite with `POST /api/members/{id}/invite`.

The member sends the invite to `POST /api/identity/accept-invite`. A successful acceptance rotates any previous member access credential and returns the new access token once.

Member access tokens create server-side sessions and always use the member's current RBAC role. Disabled or deleted members are immediately invalid.

### TOTP MFA

An authenticated member can start MFA setup with `POST /api/identity/mfa`. Amaquet returns a base32 TOTP secret and an `otpauth://` URI. Confirm setup with `PUT /api/identity/mfa` and a current six-digit code.

After confirmation, new member sessions require both the access token and `mfa_code`. Codes use HMAC-SHA-1, a 30-second step, six decimal digits, and a ±1-step verification window. `DELETE /api/identity/mfa` disables TOTP for the current member.

TOTP secrets are sensitive and are stored in the crash-safe administration state file. Protect its backups.

## One-time bootstrap

The bootstrap credential only initializes a new control plane. If startup needs to generate it, Amaquet writes the secret to `data_dir/bootstrap-token` with file mode `0600` and logs the path, not the secret.

When an existing administration state still has the untouched organization default from before the Amaquet rename, the schema migration updates that default to `Amaquet` / `amaquet`. User-customized organization names and all existing credential hashes remain unchanged.

After bootstrap creates the first `admin` API key, bootstrap authentication is disabled in persisted administration state.

## HTTP cookie sessions

`POST /api/session` exchanges a login credential for a 12-hour server-side session. The client receives an `HttpOnly`, `SameSite=Strict` cookie. HTTPS also sets the `Secure` flag.

Sessions are bounded, expired sessions are pruned, and actor validity is checked on use. API-key revocation or member revocation removes matching sessions through the revocation hook.

## RBAC

Built-in roles are `admin`, `auditor`, and `developer`. Their initial permissions are:

| Role        | Default permissions                                                                            |
| ----------- | ---------------------------------------------------------------------------------------------- |
| `admin`     | `*`                                                                                            |
| `auditor`   | `data.read`, `org.read`, `members.read`, `rbac.read`, `keys.read`, `system.read`, `audit.read` |
| `developer` | `data.read`, `data.write`, `org.read`, `members.read`, `system.read`                           |

Protocol commands map to `data.read` or `data.write`. Admin routes use permissions for organization, members, RBAC, keys, system configuration, persistence, audit, and data access. The RBAC map is editable, so these are defaults rather than permanent role definitions.

Role changes take effect on the next actor validation. Existing sessions do not retain an old role snapshot indefinitely.

## TLS

`amaquets://` uses the Amaquet TLS listener. A non-loopback authenticated protocol listener requires TLS unless the explicit insecure override is enabled.

The admin listener has independent HTTPS settings. Non-loopback plain HTTP is rejected by default.

## Ed25519 server identity

The protocol can load a separate Ed25519 identity. `HELLO` can sign a client nonce and return the public key and fingerprint. This proves possession of the configured identity key. It does not replace TLS certificate validation.

## Crash-safe control-plane state

Configuration and administration state use a temporary write, file fsync, atomic rename, and directory fsync. Administration state currently uses schema version 3 and has explicit migrations from earlier schemas; a state file created by a newer unsupported schema is rejected instead of being partially interpreted.

The in-file audit history retains the newest 5,000 events. API-key last-use timestamps are buffered and flushed to disk on a five-second loop (and at orderly state shutdown) so authentication does not force an fsync for every request.
