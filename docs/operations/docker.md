---
title: "Docker"
---

The repository contains `Dockerfile`, a loopback-only development `docker-compose.yml`, and a TLS-oriented `docker-compose.production.yml`. Both Compose files build the same image and persist `/app/data` in the `amaquet-data` volume.

Typical workflow:

```bash
docker compose build
docker compose up -d
```

The development Compose file requires `AMAQUET_BOOTSTRAP_TOKEN` before it can start:

```bash
export AMAQUET_BOOTSTRAP_TOKEN="$(openssl rand -base64 32)"
docker compose up -d --build
```

The value is injected into the container as an environment override; it is not written back to the repository configuration. The development file intentionally enables plaintext authentication inside the container because the published host ports are restricted to `127.0.0.1`. Do not reuse that profile when publishing the service remotely.

Persist `data_dir` with a volume if AOF or admin state must survive container replacement. Mount production TLS and identity keys read-only where possible.

Do not store production bootstrap tokens directly in a public Compose file. Inject secrets through the deployment platform.

After start, check the unauthenticated health endpoint and then the authenticated native listener:

```bash
curl http://127.0.0.1:13379/api/health
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_BOOTSTRAP_TOKEN" PING '{}'
```

The Compose configuration requires `AMAQUET_BOOTSTRAP_TOKEN` and enables protocol authentication, so the protocol check supplies that credential explicitly.

The image uses a Go build stage to produce four statically linked binaries, and the final Alpine image runs as the unprivileged `amaquet` user. Compose binds host ports only to `127.0.0.1`, persists `/app/data` in `amaquet-data`, enables AOF `everysec`, and uses explicit insecure-listener overrides only because Docker must bind `0.0.0.0` inside the isolated container network.

The production Compose file is a template rather than a zero-configuration launch. Before starting it, provision:

- `./certs/amaquet-server.crt` and `./certs/amaquet-server.key`;
- `./certs/amaquet-admin.crt` and `./certs/amaquet-admin.key`;
- an `AMAQUET_BOOTSTRAP_TOKEN` environment value; and
- the Ed25519 files referenced by `/app/data/keys/amaquet-public.pem` and `/app/data/keys/amaquet-private.pem`, unless those paths are removed from the production configuration.

The production image does not generate certificates or identity keys. TLS certificates must match the names used by clients, and private files must be readable by the unprivileged `amaquet` container user. Review host firewall rules before exposing either published port.
