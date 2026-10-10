#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:tuple:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"tuple\",\"options\":{\"values\":[{\"type\":\"utf8_string\",\"value\":\"alpha\"},{\"type\":\"integer\",\"value\":7}]}}" >/dev/null
key_lifecycle "$KEY" tuple

out="$(extract "$(op "$KEY" LEN '{}')")"
assert_eq "$out" 2 "LEN"

out="$(op "$KEY" GET '{"index":0}')"
assert_contains "$out" alpha "GET[0]"

key_del "$KEY"
finish tuple
