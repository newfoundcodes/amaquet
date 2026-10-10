#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:inverted_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"inverted_index\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" inverted_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"INDEX\",\"args\":{\"id\":\"d1\",\"text\":\"amaquet in-memory database\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"INDEX\",\"args\":{\"id\":\"d2\",\"text\":\"redis alternative go\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"query\":\"memory\",\"limit\":10}}")"
assert_contains "$out" d1 "SEARCH"

key_del "$KEY"
finish inverted_index
