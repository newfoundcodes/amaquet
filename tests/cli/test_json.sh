#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:json:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"json\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" json

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"value\":{\"hello\":\"world\",\"n\":1}}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" world "GET"

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"path\":\"$.n\",\"value\":42}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" 42 "GET after path SET"

key_del "$KEY"
finish json
