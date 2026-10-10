#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:hyperloglog:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"hyperloglog\",\"options\":{\"precision\":12}}" >/dev/null
key_lifecycle "$KEY" hyperloglog

for item in alice bob carol dave eve frank; do
    op "$KEY" ADD '{"value":"'"$item"'"}' >/dev/null
done

out="$(extract "$(op "$KEY" COUNT '{}')")"
assert_ge "$out" 1 "HLL estimate >= 1"

key_del "$KEY"
finish hyperloglog
