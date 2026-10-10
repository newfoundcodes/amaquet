#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
h="$(scenario_key hashcon)"
s="$(scenario_key setcon)"

create "$h" hashmap '{}'
create "$s" set '{}'

workers="${AMAQUET_CONCURRENCY:-6}"
per="$AMAQUET_SCENARIO_ITERATIONS"

pids=()
for ((w = 0; w < workers; w++)); do
    (
        for ((i = 0; i < per; i++)); do
            "$AMAQUET_CLI" -uri "$AMAQUET_URI" OP "{\"key\":\"$h\",\"operation\":\"SET\",\"args\":{\"field\":\"w${w}_$i\",\"value\":{\"type\":\"integer\",\"value\":$i}}}" >/dev/null
            "$AMAQUET_CLI" -uri "$AMAQUET_URI" OP "{\"key\":\"$s\",\"operation\":\"ADD\",\"args\":{\"value\":{\"type\":\"utf8_string\",\"value\":\"w${w}_$i\"}}}" >/dev/null
        done
    ) &
    pids+=("$!")
done

for p in "${pids[@]}"; do wait "$p"; done
expected=$((workers * per))

assert_eq "$(op "$h" KEYS '{}' | json_len)" "$expected"
assert_eq "$(op "$s" MEMBERS '{}' | json_len)" "$expected"

server_healthy
echo "PASS concurrent collections expected=$expected"
