---
title: "Troubleshooting"
---

Use these symptoms and corrective actions to diagnose common startup, authentication, persistence, TLS, and test failures.

## `authentication required`

The server has `protocol.require_auth=true` and the connection did not authenticate. Pass `amaquet-cli -api-key "$AMAQUET_API_KEY"`, use `DialWithOptions` with `DialOptions.APIKey`, or send the `AUTH` opcode. URI credentials are rejected by the bundled clients unless the legacy opt-in is enabled.

## `permission denied`

The API key authenticated but its role lacks the command's required permission. Inspect `/api/rbac` with an authorized administrator.

## `wrong value type`

The operation targeted a key whose stable type does not support that operation. Run `TYPE` and use the relevant type reference.

## Server does not start with TLS

Verify both certificate/key paths and file permissions. TLS requires both configured files.

## AOF replay fails

Check the error for invalid header, record size, JSON decode, replay error, or CRC mismatch. Make a copy of the AOF before manual diagnosis.

## Bootstrap-token file is missing

`data_dir` is also resolved from the process working directory. A token file exists only when `security.bootstrap_token` is empty at startup; `--init-config` instead embeds a generated token in the JSON file. From `bin/`, a relative `./data/bootstrap-token` is therefore `bin/data/bootstrap-token`. Check the effective configuration and working directory before assuming the token was not generated.

## Bash test fails

Run one script directly with `AMAQUET_KEEP_TMP=1` to preserve the generated test directory and `server.log`:

```bash
AMAQUET_KEEP_TMP=1 tests/bash/types/test_vector_set.sh
```
