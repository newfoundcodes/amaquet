#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:numeric_field:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"numeric_field\",\"options\":{\"name\":\"score\",\"value\":42.5}}" >/dev/null
key_lifecycle "$KEY" numeric_field

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"name\":\"score\",\"value\":99.5}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" 99.5 "GET"

key_del "$KEY"
finish numeric_field
