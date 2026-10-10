#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:lock:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"lock\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" lock

out="$(extract "$(op "$KEY" TRY_LOCK '{"owner":"tester","token":"t1","ttl_ms":5000}')")"
assert_true "$out" "TRY_LOCK"

out="$(extract "$(op "$KEY" TRY_LOCK '{"owner":"other","token":"t2","ttl_ms":1000}')")"
assert_false "$out" "TRY_LOCK while held"

out="$(extract "$(op "$KEY" UNLOCK '{"token":"t1"}')")"
assert_true "$out" "UNLOCK"

key_del "$KEY"
finish lock
