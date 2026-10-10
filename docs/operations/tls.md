---
title: "TLS and protocol identity"
---

Configure TLS for transport security and optional Ed25519 keys for stable application-level server identity.

## Amaquet TLS

Configure:

```json
{
  "protocol": {
    "tls": true,
    "tls_cert_file": "/run/secrets/amaquet.crt",
    "tls_key_file": "/run/secrets/amaquet.key"
  }
}
```

Connect with `amaquets://`. The Go client uses normal TLS server-name verification based on the URI host.

## Protocol identity

Generate a separate Ed25519 identity with `amaquet-keygen` and configure `security.public_key_file` and `security.private_key_file`.

TLS protects the transport and authenticates through certificates. The Amaquet identity signs application nonces and gives clients a stable application-level key fingerprint. Use both when you require both guarantees.
