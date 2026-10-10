#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:boolean:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"boolean\",\"value\":true}" >/dev/null
key_lifecycle "$KEY" boolean

out="$(extract "$(op "$KEY" TOGGLE '{}')")"
assert_false "$out" "TOGGLE"

key_del "$KEY"
finish boolean
