#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
for ((i = 1; i <= AMAQUET_SCENARIO_ITERATIONS; i++)); do
    k="scenario:bin:$$:$i"
    b64="$(
        python3 - "$i" "$AMAQUET_FUZZ_SEED" <<'PY'
import base64,random,sys
r=random.Random(int(sys.argv[2])+int(sys.argv[1])); n=1+(int(sys.argv[1])*97)%4096
print(base64.b64encode(bytes(r.randrange(256) for _ in range(n))).decode())
PY
    )"
    setv "$k" binary_string "\"$b64\""
    got="$(amaquet GET "{\"key\":\"$k\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["value"])')"
    assert_eq "$got" "$b64"
done

u="$(scenario_key unicode)"
val='Grüße 👋 — 日本語 — العربية — 😀 — line\nfeed — tab\tend'

amaquet SET "$(
    python3 - "$u" "$val" <<'PY'
import json,sys
print(json.dumps({"key":sys.argv[1],"type":"utf8_string","value":sys.argv[2]}))
PY
)" >/dev/null
out="$(amaquet GET "{\"key\":\"$u\"}" | python3 -c 'import json,sys;print(json.load(sys.stdin)["value"])')"

assert_eq "$out" "$val"
server_healthy

echo "PASS binary/unicode roundtrip"
