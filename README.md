<div align="center">
  <img src="assets/amaquet-logo.png" width="128" />
  <h1>Amaquet</h1>
  <p>
    <a href="https://github.com/newfoundcodes/amaquet/actions"><img src="https://github.com/newfoundcodes/amaquet/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a> <a href="https://github.com/newfoundcodes/amaquet/actions"><img src="https://github.com/newfoundcodes/amaquet/actions/workflows/packages.yml/badge.svg" alt="Packaging Status"></a> <a href="https://github.com/newfoundcodes/amaquet/actions"><img src="https://github.com/newfoundcodes/amaquet/actions/workflows/soak.yml/badge.svg" alt="Soaking Status"></a>
  </p>
</div>

Amaquet is a standalone, Redis-like in-memory database written in Go. It does not use Redis or another database as its storage engine. Amaquet is Redis-like in purpose, but it is not Redis or RESP compatible.

It provides a custom binary-framed protocol with JSON command payloads, `amaquet://` plaintext TCP connections, and `amaquets://` TLS connections. The server includes a strongly typed data model, a 256-shard concurrent keyspace, TTL and per-key versions, optional CRC-protected append-only persistence, API-key authentication, RBAC, and a separate HTTP administration API.

By default, Amaquet binds to loopback, requires native protocol authentication, uses ports 13378 and 13379, and does not enable TLS unless configured.

<p align="center">
  <a href="https://amaquet.newfoundcodes.com">
    <img src="https://img.shields.io/badge/Visit%20Documentation-000000?style=for-the-badge&logo=astro&logoColor=ffffff" alt="Visit Documentation">
  </a>
</p>

## Repository layout

```text
cmd/amaquet/              Amaquet server
cmd/amaquet-cli/          command-line Amaquet client
cmd/amaquet-keygen/       Ed25519 identity key generator
cmd/amaquet-restore/      Verified offline checkpoint restore tool
internal/core/         sharded keyspace and stable data-type registry
internal/types/        concrete data structures
internal/protocol/     Amaquet frame format and URI parser
internal/server/       TCP server, dispatcher, factories
internal/admin/        RBAC/control-plane state and HTTP API
internal/security/     Ed25519 server identity
internal/persistence/  optional append-only journal
pkg/amaquet/              reusable Go client
docs/                  Astro/Starlight documentation site and content
tests/                 Go integration, per-type fuzzers, developer scenarios, security/AOF fuzzing
```

## License

Amaquet is licensed under the GNU Affero General Public License, version 3 (AGPLv3). See [LICENSE](LICENSE) for the complete license text.
