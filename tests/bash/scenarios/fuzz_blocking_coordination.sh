#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
q="$(scenario_key blocking)"
b="$(scenario_key barrier)"

create "$q" blocking_queue '{}'
create "$b" barrier '{"target":3}'

out="${AMAQUET_TEST_TMP:-/tmp}/blocking-$$.out"
"$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout 5s OP "{\"key\":\"$q\",\"operation\":\"DEQUEUE\",\"args\":{}}" >"$out" &
pid=$!

sleep .05
op "$q" ENQUEUE '{"value":{"type":"utf8_string","value":"wake"}}' >/dev/null
wait "$pid"

assert_contains "$(cat "$out")" wake
pids=()

for i in 1 2 3; do
    "$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout 5s OP "{\"key\":\"$b\",\"operation\":\"WAIT\",\"args\":{}}" >"${out}.$i" &
    pids+=("$!")
done

for p in "${pids[@]}"; do wait "$p"; done
for i in 1 2 3; do assert_true "$(cat "${out}.$i" | json_extract x)"; done

rm -f "$out" "${out}."*
server_healthy

echo "PASS blocking queue/barrier coordination"
