#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
k="$(scenario_key compression)"
tmp="${AMAQUET_TEST_TMP:-/tmp}/compression-$$.json"

printf '{"key":"%s","type":"utf8_string","value":"abc"}\n' "$k" >"$tmp"
"$AMAQUET_CLI" -uri "$AMAQUET_URI" SET "@$tmp" >/dev/null
assert_false "$(amaquet MEMORY "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("compressed",False))')"

python3 - "$k" >"$tmp" <<'PY'
import json,sys
print(json.dumps({"key":sys.argv[1],"type":"utf8_string","value":"A"*131072}))
PY
"$AMAQUET_CLI" -uri "$AMAQUET_URI" SET "@$tmp" >/dev/null

mem="$(amaquet MEMORY "{\"key\":\"$k\"}")"
assert_true "$(printf '%s' "$mem" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("compressed",False))')"

saving="$(printf '%s' "$mem" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("savings_bytes",0))')"
((saving > 0)) || die "compression did not save bytes"
gotlen="$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)["value"]))')"
assert_eq "$gotlen" 131072

printf '{"key":"%s","type":"utf8_string","value":"abc"}\n' "$k" >"$tmp"
"$AMAQUET_CLI" -uri "$AMAQUET_URI" SET "@$tmp" >/dev/null
assert_false "$(amaquet MEMORY "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("compressed",False))')"

rm -f "$tmp"
server_healthy

echo "PASS compression transitions"
