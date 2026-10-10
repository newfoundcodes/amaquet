#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:float:$$:$RANDOM"

start_server

cli SET "{\"key\":\"$KEY\",\"type\":\"float\",\"value\":3.14159}" >/dev/null
key_lifecycle "$KEY" float

out="$(extract "$(op "$KEY" ADD '{"delta":0.5}')")"
assert_eq "$out" 3.64159 "ADD"

key_del "$KEY"
finish float
