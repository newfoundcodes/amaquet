#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:priority_queue:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"priority_queue\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" priority_queue

op "$KEY" ENQUEUE '{"value":{"type":"utf8_string","value":"low"},"priority":1}' >/dev/null
op "$KEY" ENQUEUE '{"value":{"type":"utf8_string","value":"high"},"priority":10}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 2 "LEN"

op "$KEY" DEQUEUE '{}' >/dev/null

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 1 "LEN after DEQUEUE"

key_del "$KEY"
finish priority_queue
