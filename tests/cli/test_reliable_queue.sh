#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:reliable_queue:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"reliable_queue\",\"options\":{\"max_attempts\":3}}" >/dev/null
key_lifecycle "$KEY" reliable_queue

op "$KEY" ENQUEUE '{"id":"job1","value":{"type":"utf8_string","value":"work"}}' >/dev/null

out="$(op "$KEY" CLAIM '{"lease_ms":5000}')"
assert_contains "$out" job1 "CLAIM"

out="$(extract "$(op "$KEY" ACK '{"id":"job1"}')")"
assert_true "$out" "ACK"

key_del "$KEY"
finish reliable_queue
