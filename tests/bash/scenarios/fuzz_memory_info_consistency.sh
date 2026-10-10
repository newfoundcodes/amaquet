#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
p="scenario:info:$$"
before="$(amaquet INFO '{}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["keys"])')"

for i in 1 2 3 4 5; do setv "$p:$i" integer "$i"; done

after="$(amaquet INFO '{}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["keys"])')"
((after >= before + 5)) || die "INFO key count did not increase: before=$before after=$after"

for i in 1 2 3 4 5; do
    m="$(amaquet MEMORY "{\"key\":\"$p:$i\"}")"
    assert_contains "$m" '"type": "integer"'
done

amaquet DEL "{\"keys\":[\"$p:1\",\"$p:2\",\"$p:3\",\"$p:4\",\"$p:5\"]}" >/dev/null

server_healthy
echo "PASS memory/info consistency"
