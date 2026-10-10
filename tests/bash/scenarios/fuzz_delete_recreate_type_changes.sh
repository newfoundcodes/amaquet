#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key recreate)"
types=(integer utf8_string list set hashmap atomic_counter)

for t in "${types[@]}"; do
    amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null 2>&1 || true
    setup_type "$k" "$t"
    assert_eq "$(amaquet TYPE "{\"key\":\"$k\"}" | json_extract x)" "$t"
    run_type_semantics "$k" "$t"
done

amaquet DEL "{\"keys\":[\"$k\"]}" >/dev/null
server_healthy

echo "PASS delete/recreate type changes"
