---
title: "Connection URIs"
---

Use Amaquet connection URIs to select a protocol transport and identify the target server endpoint.

## Syntax

An Amaquet connection URI specifies the transport scheme, host, and optional port for one server keyspace.

```text
amaquet://host[:port]
amaquets://host[:port]
```

The default port is `13378`.

Examples:

```text
amaquet://127.0.0.1:13378
amaquet://example.com
amaquets://db.example.com:13378
amaquet://user:secret@127.0.0.1:13378
amaquet://127.0.0.1:13378?api_key=amaquet_...
```

The parser recognizes credentials in URI user-info and the `api_key` query parameter for legacy compatibility. If user-info contains a password, the password becomes the API key; with a username only, the username does. `api_key` overrides user-info.

The public Go client and CLI reject URI-embedded credentials by default. Pass `DialOptions.APIKey` or `amaquet-cli -api-key` instead. Enabling `AllowURISecrets`/`-allow-uri-secrets` is explicit opt-in because URLs can leak through logs, shell history, process listings, and telemetry.

An empty path or a bare trailing `/` is accepted. Any path component after that slash is rejected because Amaquet has one keyspace and does not implement logical database names.
