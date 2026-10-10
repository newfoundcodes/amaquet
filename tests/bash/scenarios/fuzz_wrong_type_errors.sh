#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key wrongtype)"

setv "$k" integer 7
for bad in PUSH_BACK ENQUEUE SEARCH SETBIT INDEX ACQUIRE; do assert_command_fails op "$k" "$bad" '{}'; done

assert_eq "$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["value"])')" 7
amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null

k="$(scenario_key listwrong)"
create "$k" list '{}'
op "$k" PUSH_BACK '{"value":{"type":"utf8_string","value":"ok"}}' >/dev/null

assert_command_fails op "$k" ADD '{"delta":1}'
assert_eq "$(op "$k" LEN '{}' | json_extract x)" 1

server_healthy
echo "PASS wrong-type errors"
