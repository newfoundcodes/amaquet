#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$HERE/lib/common.sh"

: "${AMAQUET_TEST_JOBS:=4}"
: "${AMAQUET_FUZZ_ITERATIONS:=25}"

export AMAQUET_FUZZ_ITERATIONS
[[ "$AMAQUET_TEST_JOBS" =~ ^[1-9][0-9]*$ ]] || die "AMAQUET_TEST_JOBS must be a positive integer"

start_test_server

work="$(mktemp -d -t amaquet-suite.XXXXXX)"
cleanup_suite() { rm -rf "$work"; }

# common.sh already owns server cleanup through its EXIT trap. Chain only our temp cleanup.
trap 'cleanup_suite; stop_test_server' EXIT INT TERM

fail=0
count=0

batch_pids=()
batch_logs=()
batch_names=()

wait_batch() {
    local i pid log name
    for i in "${!batch_pids[@]}"; do
        pid="${batch_pids[$i]}"
        log="${batch_logs[$i]}"
        name="${batch_names[$i]}"
        printf '\n==> %s (%s iterations)\n' "$name" "$AMAQUET_FUZZ_ITERATIONS"

        if wait "$pid"; then cat "$log"; else
            fail=$((fail + 1))
            cat "$log" >&2
        fi
    done

    batch_pids=()
    batch_logs=()
    batch_names=()
}

for test in "$HERE"/fuzz/fuzz_*.sh; do
    count=$((count + 1))
    name="$(basename "$test")"
    log="$work/$name.log"
    (AMAQUET_URI="$AMAQUET_URI" AMAQUET_BIN="$AMAQUET_BIN" AMAQUET_CLI="$AMAQUET_CLI" "$test" >"$log" 2>&1) &

    batch_pids+=("$!")
    batch_logs+=("$log")
    batch_names+=("$name")
    if ((${#batch_pids[@]} >= AMAQUET_TEST_JOBS)); then wait_batch; fi
done

if ((${#batch_pids[@]})); then wait_batch; fi

printf '\nRan %d type fuzzers; failures: %d\n' "$count" "$fail"
[[ "$fail" -eq 0 ]]
