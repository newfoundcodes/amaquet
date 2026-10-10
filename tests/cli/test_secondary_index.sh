#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:secondary_index:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"secondary_index\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" secondary_index

cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"field\":\"status\",\"value\":\"active\",\"id\":\"u1\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"field\":\"status\",\"value\":\"active\",\"id\":\"u2\"}}" >/dev/null
cli OP "{\"key\":\"$KEY\",\"operation\":\"ADD\",\"args\":{\"field\":\"status\",\"value\":\"inactive\",\"id\":\"u3\"}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"LOOKUP\",\"args\":{\"field\":\"status\",\"value\":\"active\"}}")"
assert_contains "$out" u1 "LOOKUP u1"
assert_contains "$out" u2 "LOOKUP u2"

key_del "$KEY"
finish secondary_index
