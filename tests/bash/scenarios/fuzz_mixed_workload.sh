#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
p="scenario:mixed:$$"

create "$p:list" list '{}'
create "$p:set" set '{}'
create "$p:counter" atomic_counter '{"initial":0}'
create "$p:hash" hashmap '{}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS * 5; i++)); do case $((RANDOM % 8)) in
    0) amaquet SET "{\"key\":\"$p:s$((i % 7))\",\"type\":\"integer\",\"value\":$i}" >/dev/null ;;
    1) op "$p:list" PUSH_BACK "{\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null ;;
    2) op "$p:set" ADD "{\"value\":{\"type\":\"integer\",\"value\":$((i % 13))}}" >/dev/null ;;
    3) op "$p:counter" ADD '{"delta":1}' >/dev/null ;;
    4) op "$p:hash" SET "{\"field\":\"f$((i % 11))\",\"value\":{\"type\":\"integer\",\"value\":$i}}" >/dev/null ;;
    5) amaquet KEYS "{\"prefix\":\"$p\",\"limit\":100}" >/dev/null ;;
    6) amaquet INFO '{}' >/dev/null ;;
    7) amaquet GET "{\"key\":\"$p:s$((i % 7))\"}" >/dev/null 2>&1 || true ;; esac done

server_healthy
assert_eq "$(amaquet TYPE "{\"key\":\"$p:counter\"}" | json_extract x)" atomic_counter

echo "PASS mixed workload seed=$AMAQUET_FUZZ_SEED"
