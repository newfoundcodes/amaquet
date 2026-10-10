#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:timeseries:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"timeseries\",\"options\":{\"labels\":{\"service\":\"api\",\"env\":\"test\"},\"retention_ms\":3600000}}" >/dev/null
key_lifecycle "$KEY" timeseries

now_ms=$(($(date +%s) * 1000))
op "$KEY" ADD '{"timestamp":'"$now_ms"',"value":10}' >/dev/null
op "$KEY" ADD '{"timestamp":'"$((now_ms + 1000))"',"value":20}' >/dev/null
op "$KEY" ADD '{"timestamp":'"$((now_ms + 2000))"',"value":30}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 3 "LEN"

key_del "$KEY"
finish timeseries
