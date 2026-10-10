---
title: "Production deployment"
---

Use this guidance to deploy Amaquet as a secure, monitored single-node stateful service.

## Recommended topology

Run Amaquet as a single-node stateful service and keep both listeners as narrow as the application permits. These controls address the implementation's in-memory data model and its separate native-protocol and administration surfaces.

- Bind the Amaquet listener only on interfaces required by applications.
- Enable TLS and API-key authentication.
- Keep the admin listener private or behind an authenticated reverse proxy.
- Put `data_dir`, AOF, identity private key, and TLS private key on protected storage.
- Set OS file-descriptor limits above expected connection counts.
- Monitor process RSS because primary data resides in memory.

## Restart behavior

The Go server handles `SIGINT` and `SIGTERM`, gracefully shuts down the admin HTTP server, closes the Amaquet listener, waits for active connection goroutines, and closes the engine. AOF close flushes buffered records.

## Stateless versus durable mode

With AOF disabled, a restart starts with an empty data keyspace while administrative state may remain in `admin.json`.

With AOF enabled, replay reconstructs journaled data operations before listeners begin serving.

## Process supervision

Use systemd, Docker, Kubernetes, or another supervisor. Configure restart policies carefully: repeated restart loops on AOF corruption should alert an operator instead of masking the underlying file problem.

## Migrating an existing installation to Amaquet

Treat the product rename as an application upgrade and back up the administration state and AOF before replacing binaries.

1. Stop the existing server so its journal and administration state are quiescent.
2. Deploy the `amaquet`, `amaquet-cli`, `amaquet-keygen`, and `amaquet-restore` binaries and rename the JSON configuration file to `amaquet.json`.
3. Update configuration paths, service units, scripts, and containers to use the `AMAQUET_*` environment variables, `amaquet://` or `amaquets://` endpoint scheme, and current binary names.
4. Update HTTP automation to use `X-Amaquet-Admin-Token`, monitoring queries to use `amaquet_*` metrics, and Go applications to import `github.com/newfoundcodes/amaquet/pkg/amaquet`.
5. Keep the existing `data_dir` and AOF path pointed at the real persisted files for the first start. Amaquet recognizes pre-rename v1 and v2 journal signatures and atomically rewrites them to `AMQTAOF2`. Administration schema migration changes only the untouched historical default organization name; customized organization data and credential hashes are preserved.
6. Start Amaquet, verify `/api/health`, `/api/ready`, authenticated `/metrics`, and an application read before removing the backup.

The protocol magic is now `AMQT`; endpoint schemes and the four-byte wire magic are not negotiated aliases. Clients and servers therefore need to be upgraded together. Existing API-key and member secrets remain valid because authentication is based on their persisted hashes, not their display prefixes.

Compose files pin the project name to `amaquet`. If upgrading a Compose deployment, explicitly attach or copy data from the previous named volume into `amaquet-data`; changing the Compose project or volume name does not move data. The container runtime user is also named `amaquet`, so ensure mounted files are readable and writable by its configured UID/GID before startup.
