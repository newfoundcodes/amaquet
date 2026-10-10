#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:duration:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"duration\",\"value\":\"5s\"}" >/dev/null
key_lifecycle "$KEY" duration

out="$(extract "$(op "$KEY" ADD '{"delta_ms":1000}')")"
assert_contains "$out" 6s "ADD"

key_del "$KEY"
finish duration
