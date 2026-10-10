#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:semaphore:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"semaphore\",\"options\":{\"limit\":2}}" >/dev/null
key_lifecycle "$KEY" semaphore

out="$(extract "$(op "$KEY" TRY_ACQUIRE '{}')")"
assert_true "$out" "TRY_ACQUIRE 1"

out="$(extract "$(op "$KEY" TRY_ACQUIRE '{}')")"
assert_true "$out" "TRY_ACQUIRE 2"

out="$(extract "$(op "$KEY" TRY_ACQUIRE '{}')")"
assert_false "$out" "TRY_ACQUIRE 3 (limit reached)"

out="$(op "$KEY" STATS '{}')"
assert_contains "$out" limit "STATS"

op "$KEY" RELEASE '{}' >/dev/null

key_del "$KEY"
finish semaphore
