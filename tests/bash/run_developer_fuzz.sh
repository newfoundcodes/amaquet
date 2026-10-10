#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
: "${AMAQUET_SCENARIO_ITERATIONS:=10}"
: "${AMAQUET_FUZZ_ITERATIONS:=2}"
: "${AMAQUET_FRAME_FUZZ_ITERATIONS:=100}"
: "${AMAQUET_FUZZ_SEED:=1337}"

export AMAQUET_SCENARIO_ITERATIONS AMAQUET_FUZZ_ITERATIONS AMAQUET_FRAME_FUZZ_ITERATIONS AMAQUET_FUZZ_SEED

"$HERE/run_scenarios.sh"
"$HERE/run_isolated_fuzz.sh"
"$HERE/run_fuzz.sh"
"$HERE/run_protocol_fuzz.sh"

printf '\nAll Amaquet developer fuzz layers passed. seed=%s scenario_iterations=%s type_iterations=%s frame_iterations=%s\n' "$AMAQUET_FUZZ_SEED" "$AMAQUET_SCENARIO_ITERATIONS" "$AMAQUET_FUZZ_ITERATIONS" "$AMAQUET_FRAME_FUZZ_ITERATIONS"
