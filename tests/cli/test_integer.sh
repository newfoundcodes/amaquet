#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:integer:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"integer\",\"value\":42}" >/dev/null
key_lifecycle "$KEY" integer

out="$(extract "$(op "$KEY" ADD '{"delta":8}')")"
assert_eq "$out" 50 "ADD"

key_del "$KEY"
finish integer
