#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:lease:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"lease\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" lease

out="$(extract "$(op "$KEY" ACQUIRE '{"owner":"tester","token":"t1","ttl_ms":5000}')")"
assert_true "$out" "ACQUIRE"

out="$(extract "$(op "$KEY" HELD '{}')")"
assert_true "$out" "HELD"

out="$(extract "$(op "$KEY" RELEASE '{"token":"t1"}')")"
assert_true "$out" "RELEASE"

out="$(extract "$(op "$KEY" HELD '{}')")"
assert_false "$out" "NOT HELD after release"

key_del "$KEY"
finish lease
