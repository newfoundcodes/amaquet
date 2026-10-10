#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:rate_limit_bucket:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"rate_limit_bucket\",\"options\":{\"capacity\":5,\"refill_per_second\":2}}" >/dev/null
key_lifecycle "$KEY" rate_limit_bucket

out="$(extract "$(op "$KEY" ALLOW '{"cost":1}')")"
assert_true "$out" "ALLOW cost=1"

out="$(extract "$(op "$KEY" REMAINING '{}')")"
assert_nonempty "$out" "REMAINING"

# Drain the bucket
for i in $(seq 1 5); do
    op "$KEY" ALLOW '{"cost":1}' >/dev/null || true
done

out="$(op "$KEY" REMAINING '{}')"
assert_nonempty "$out" "REMAINING value"

key_del "$KEY"
finish rate_limit_bucket
