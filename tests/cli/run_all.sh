#!/usr/bin/env bash
# Usage:
#   ./tests/cli/run_all.sh
# Env overrides:
#   AMAQUET_CLI_JOBS   — parallel workers (default: 4)
#   AMAQUET_URI        — use external server instead of spawning one
#   AMAQUET_CLI        — path to amaquet-cli binary
#   AMAQUET_BIN        — path to amaquet server binary
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$HERE/lib.sh"

: "${AMAQUET_CLI_JOBS:=4}"
[[ "$AMAQUET_CLI_JOBS" =~ ^[1-9][0-9]*$ ]] || fail "AMAQUET_CLI_JOBS must be a positive integer"

start_server

work="$(mktemp -d -t amaquet-cli-suite.XXXXXX)"
_cleanup_suite() { rm -rf "$work"; }
trap '_cleanup_suite; _stop_server' EXIT INT TERM

fail_count=0
total=0

batch_pids=()
batch_logs=()
batch_names=()

wait_batch() {
    local i pid log name
    for i in "${!batch_pids[@]}"; do
        pid="${batch_pids[$i]}"
        log="${batch_logs[$i]}"
        name="${batch_names[$i]}"

        printf '\n==> %s\n' "$name"
        if wait "$pid"; then
            cat "$log"
        else
            fail_count=$((fail_count + 1))
            cat "$log" >&2
        fi
    done

    batch_pids=()
    batch_logs=()
    batch_names=()
}

for test_script in "$HERE"/test_*.sh; do
    total=$((total + 1))
    name="$(basename "$test_script")"
    log="$work/$name.log"

    (AMAQUET_URI="$URI" AMAQUET_CLI="$CLI" AMAQUET_BIN="$SRV" \
        "$test_script" >"$log" 2>&1) &

    batch_pids+=("$!")
    batch_logs+=("$log")
    batch_names+=("$name")

    if ((${#batch_pids[@]} >= AMAQUET_CLI_JOBS)); then
        wait_batch
    fi
done

if ((${#batch_pids[@]})); then wait_batch; fi

printf '\nRan %d CLI type tests; failures: %d\n' "$total" "$fail_count"
[[ "$fail_count" -eq 0 ]]
