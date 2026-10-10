---
title: "CLI reference"
description: "Use the Amaquet command-line tools for native protocol commands, API-key authentication, server identity generation, and verified checkpoint restoration."
---

This reference describes the Amaquet client, server, identity-generation, and checkpoint-restore executables.

## `amaquet-cli`

`amaquet-cli` sends Amaquet commands.

```bash
amaquet-cli \
  -uri amaquets://db.example.com:13378 \
  -api-key "$AMAQUET_API_KEY" \
  GET '{"key":"example"}'
```

For large JSON arguments, use a file or stdin instead of a shell argument:

```bash
amaquet-cli SET @request.json
cat request.json | amaquet-cli SET -
```

URI-embedded secrets are disabled by default. Pass the secret with `-api-key`.

| Flag                 | Default                     | Meaning                                                      |
| -------------------- | --------------------------- | ------------------------------------------------------------ |
| `-uri`               | `amaquet://127.0.0.1:13378` | `amaquet://` or `amaquets://` endpoint with no database path |
| `-api-key`           | empty                       | API key sent with the protocol `AUTH` opcode                 |
| `-allow-uri-secrets` | `false`                     | explicitly accept legacy URI user-info/query credentials     |
| `-timeout`           | `30s`                       | dial and request context deadline                            |

The first positional argument is the command name. The optional second argument is a JSON object, `@path` to read that object from a file, or `-` to read it from standard input. JSON numbers are decoded with `UseNumber`, avoiding an automatic conversion to `float64` before transmission. Output is pretty-printed when it is valid JSON. Usage/input errors exit with status 2; dial/command errors exit with status 1.

## `amaquet`

The server executable accepts the following startup, initialization, and version-reporting flags.

| Flag           | Default          | Meaning                                                                        |
| -------------- | ---------------- | ------------------------------------------------------------------------------ |
| `-config`      | `./amaquet.json` | JSON configuration path                                                        |
| `-init-config` | `false`          | save defaults with a generated bootstrap token, print the token once, and exit |
| `-version`     | `false`          | print the server version and exit                                              |

Normal startup loads JSON if it exists, overlays supported environment variables, validates the result, creates `data_dir`, replays the AOF when enabled, then starts the admin and Amaquet listeners. If neither JSON nor the environment supplies a bootstrap token, startup reuses or creates `data_dir/bootstrap-token` with mode `0600`.

## `amaquet-keygen`

Generate the optional Ed25519 server identity:

```bash
amaquet-keygen \
  --public ./keys/amaquet-public.pem \
  --private ./keys/amaquet-private.pem
```

`-public` defaults to `amaquet-public.pem`; `-private` defaults to `amaquet-private.pem`. Existing targets are replaced. The command prints the public-key fingerprint after saving the PEM files.

## `amaquet-restore`

Restore a verified AOF checkpoint while Amaquet is stopped:

```bash
amaquet-restore \
  -source /backup/checkpoint.aof \
  -target /var/lib/amaquet/amaquet.aof
```

The tool validates the source and copied destination before the atomic swap. It maintains a temporary rollback file during replacement.

Both `-source` and `-target` are required. Run the restore command only while the target Amaquet process is stopped.
