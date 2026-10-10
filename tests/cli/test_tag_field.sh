#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:tag_field:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"tag_field\",\"options\":{\"name\":\"tags\",\"values\":[\"go\",\"database\"]}}" >/dev/null
key_lifecycle "$KEY" tag_field

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"name\":\"tags\",\"values\":[\"a\",\"b\",\"c\"]}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" a "GET"
assert_contains "$out" c "GET"

key_del "$KEY"
finish tag_field
