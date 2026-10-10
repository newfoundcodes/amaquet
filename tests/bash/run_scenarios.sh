#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$HERE/lib/scenario.sh"

: "${AMAQUET_TEST_JOBS:=2}"
: "${AMAQUET_SCENARIO_ITERATIONS:=10}"
: "${AMAQUET_FUZZ_SEED:=1337}"

export AMAQUET_SCENARIO_ITERATIONS AMAQUET_FUZZ_SEED
[[ "$AMAQUET_TEST_JOBS" =~ ^[1-9][0-9]*$ ]] || die "AMAQUET_TEST_JOBS must be positive"

start_test_server

work="$(mktemp -d -t amaquet-scenarios.XXXXXX)"
fail=0
count=0

pids=()
logs=()
names=()

cleanup_suite() { rm -rf "$work"; }

trap 'cleanup_suite; stop_test_server' EXIT INT TERM

wait_batch() {
    local i
    for i in "${!pids[@]}"; do
        printf '\n==> %s\n' "${names[$i]}"
        if wait "${pids[$i]}"; then cat "${logs[$i]}"; else
            fail=$((fail + 1))
            cat "${logs[$i]}" >&2
        fi
    done

    pids=()
    logs=()
    names=()
}

run_shared() {
    local test="$1" name log

    count=$((count + 1))
    name="$(basename "$test")"
    log="$work/$name.log"

    (AMAQUET_URI="$AMAQUET_URI" AMAQUET_BIN="$AMAQUET_BIN" AMAQUET_CLI="$AMAQUET_CLI" AMAQUET_SCENARIO_ITERATIONS="$AMAQUET_SCENARIO_ITERATIONS" AMAQUET_FUZZ_SEED="$AMAQUET_FUZZ_SEED" "$test" >"$log" 2>&1) &

    pids+=("$!")
    logs+=("$log")
    names+=("$name")

    if ((${#pids[@]} >= AMAQUET_TEST_JOBS)); then wait_batch; fi
}

for test in "$HERE"/scenarios/fuzz_*.sh; do run_shared "$test"; done
for test in "$HERE"/protocol/fuzz_multiplex_request_ids.sh "$HERE"/protocol/fuzz_partial_writes.sh "$HERE"/protocol/fuzz_oversized_declared_payload.sh "$HERE"/protocol/fuzz_reconnect_after_malformed.sh "$HERE"/protocol/fuzz_pubsub_delivery.sh; do run_shared "$test"; done

if ((${#pids[@]})); then wait_batch; fi

printf '\nRan %d shared developer scenarios; failures: %d\n' "$count" "$fail"
[[ "$fail" -eq 0 ]]
