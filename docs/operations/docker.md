---
title: "Docker"
---

The repository contains `Dockerfile`, a loopback-only development `docker-compose.yml`, and a TLS-oriented `docker-compose.production.yml`. Both Compose files build the same image and persist `/app/data` in the `amaquet-data` volume.

Typical workflow:

```bash
docker compose build
docker compose up -d
```

Persist `data_dir` with a volume if AOF or admin state must survive container replacement. Mount production TLS and identity keys read-only where possible.

Do not store production bootstrap tokens directly in a public Compose file. Inject secrets through the deployment platform.

After start, check:

```bash
./bin/amaquet-cli -uri amaquet://127.0.0.1:13378 \
  -api-key "$AMAQUET_BOOTSTRAP_TOKEN" PING '{}'
curl http://127.0.0.1:13379/api/health
```

The Compose configuration requires `AMAQUET_BOOTSTRAP_TOKEN` and enables protocol authentication, so the protocol check supplies that credential explicitly.

The image uses a Go build stage to produce four statically linked binaries, and the final Alpine image runs as the unprivileged `amaquet` user. Compose binds host ports only to `127.0.0.1`, persists `/app/data` in `amaquet-data`, enables AOF `everysec`, and uses explicit insecure-listener overrides only because Docker must bind `0.0.0.0` inside the isolated container network.

The production Compose file is a template rather than a zero-configuration launch. It publishes both ports on the host's configured interfaces, enables TLS for both listeners, and expects the four certificate/key files under `./certs`. Its configuration also names Ed25519 identity files under `/app/data/keys`, so generate those files in the persistent volume or change the paths before first startup. Review host firewall rules and certificate names before exposing either published port.
