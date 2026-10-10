#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:flat_vector_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"flat_vector_index\",\"options\":{\"dim\":3,\"metric\":\"cosine\"}}" >/dev/null
key_lifecycle "$KEY" flat_vector_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v1\",\"vector\":[1,0,0],\"metadata\":{}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v2\",\"vector\":[0,0,1],\"metadata\":{}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"vector\":[1,0,0],\"k\":1}}")"
assert_contains "$out" v1 "SEARCH"

key_del "$KEY"
finish flat_vector_index
