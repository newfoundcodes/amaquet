#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:gauge_series:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"gauge_series\",\"options\":{\"labels\":{\"metric\":\"cpu\"}}}" >/dev/null
key_lifecycle "$KEY" gauge_series

op "$KEY" SET '{"timestamp":1000,"value":42.5}' >/dev/null
op "$KEY" SET '{"timestamp":2000,"value":55.0}' >/dev/null

out="$(op "$KEY" LAST '{}')"
assert_contains "$out" '"value": 55' "LAST"

key_del "$KEY"
finish gauge_series
