#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:hnsw_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"hnsw_index\",\"options\":{\"dim\":3,\"metric\":\"cosine\",\"m\":8,\"ef_search\":32}}" >/dev/null
key_lifecycle "$KEY" hnsw_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v1\",\"vector\":[1,0,0],\"metadata\":{}}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"id\":\"v2\",\"vector\":[0,1,0],\"metadata\":{}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SEARCH\",\"args\":{\"vector\":[1,0,0],\"k\":1}}")"
assert_contains "$out" v1 "SEARCH"

key_del "$KEY"
finish hnsw_index
