#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:matrix:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"matrix\",\"options\":{\"rows\":2,\"cols\":2,\"dtype\":\"float64\",\"data\":[1,2,3,4]}}" >/dev/null
key_lifecycle "$KEY" matrix

out="$(extract "$(op "$KEY" GET '{"indices":[1,1]}')")"
assert_eq "$out" 4 "GET[1,1]"

op "$KEY" SET '{"indices":[0,1],"value":9}' >/dev/null

out="$(extract "$(op "$KEY" GET '{"indices":[0,1]}')")"
assert_eq "$out" 9 "GET after SET"

key_del "$KEY"
finish matrix
