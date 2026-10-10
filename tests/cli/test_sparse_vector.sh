#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:sparse_vector:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"sparse_vector\",\"value\":{\"dim\":8,\"values\":{\"1\":0.5,\"5\":1.25}}}" >/dev/null
key_lifecycle "$KEY" sparse_vector

out="$(cli GET "{\"key\":\"$KEY\"}")"
assert_contains "$out" '"type": "sparse_vector"' "GET"

key_del "$KEY"
finish sparse_vector
