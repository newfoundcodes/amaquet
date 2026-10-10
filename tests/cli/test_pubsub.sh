#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:pubsub:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"pubsub\",\"options\":{}}" >/dev/null
key_lifecycle "$KEY" pubsub

# PUBLISH returns subscriber count (0 since no active subscribers)
out="$(extract "$(op "$KEY" PUBLISH '{"channel":"events","value":{"type":"utf8_string","value":"ping"}}')")"
assert_eq "$out" 0 "PUBLISH count"

key_del "$KEY"
finish pubsub
