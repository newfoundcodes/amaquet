#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:t_digest:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"t_digest\",\"options\":{\"compression\":100}}" >/dev/null
key_lifecycle "$KEY" t_digest

for n in 1 2 3 4 5 6 7 8 9 10; do
    op "$KEY" ADD '{"value":'"$n"'}' >/dev/null
done

out="$(extract "$(op "$KEY" QUANTILE '{"q":0.5}')")"
assert_nonempty "$out" "p50 estimate"
assert_ge "$out" 1 "p50 >= 1"

key_del "$KEY"
finish t_digest
