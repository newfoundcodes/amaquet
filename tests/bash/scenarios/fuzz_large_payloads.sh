#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
tmp="${AMAQUET_TEST_TMP:-/tmp}/large-$$.json"

if [[ "${AMAQUET_SCENARIO_SMOKE:-0}" == 1 ]]; then sizes=(1024 65536 262144); else sizes=(1024 65536 524288 1048576 2097152); fi

for size in "${sizes[@]}"; do
    k="scenario:large:$$:$size"
    python3 - "$k" "$size" >"$tmp" <<'PY'
import json,sys
k,n=sys.argv[1],int(sys.argv[2]); print(json.dumps({"key":k,"type":"utf8_string","value":"X"*n}))
PY
    "$AMAQUET_CLI" -uri "$AMAQUET_URI" -timeout 15s SET "@$tmp" >/dev/null
    got="$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["value"]))')"
    assert_eq "$got" "$size"
done

rm -f "$tmp"
server_healthy

echo "PASS bounded large payloads"
