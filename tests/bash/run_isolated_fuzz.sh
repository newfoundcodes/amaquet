#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

: "${AMAQUET_SCENARIO_ITERATIONS:=10}"
: "${AMAQUET_FUZZ_SEED:=1337}"

export AMAQUET_SCENARIO_ITERATIONS AMAQUET_FUZZ_SEED
fail=0
count=0

for test in "$HERE"/security/fuzz_*.sh "$HERE"/persistence/fuzz_*.sh; do
    count=$((count + 1))
    printf '\n==> %s\n' "$(basename "$test")"
    if env -u AMAQUET_URI AMAQUET_SCENARIO_ITERATIONS="$AMAQUET_SCENARIO_ITERATIONS" AMAQUET_FUZZ_SEED="$AMAQUET_FUZZ_SEED" "$test"; then :; else fail=$((fail + 1)); fi
done

printf '\nRan %d isolated security/persistence fuzzers; failures: %d\n' "$count" "$fail"
[[ "$fail" -eq 0 ]]
