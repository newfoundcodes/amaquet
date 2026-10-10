#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:tensor:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"tensor\",\"options\":{\"shape\":[2,2,2],\"dtype\":\"float64\",\"data\":[0,1,2,3,4,5,6,7]}}" >/dev/null
key_lifecycle "$KEY" tensor

out="$(extract "$(op "$KEY" GET '{"indices":[1,1,1]}')")"
assert_eq "$out" 7 "GET[1,1,1]"

op "$KEY" SET '{"indices":[0,0,0],"value":99}' >/dev/null

out="$(extract "$(op "$KEY" GET '{"indices":[0,0,0]}')")"
assert_eq "$out" 99 "GET after SET"

key_del "$KEY"
finish tensor
