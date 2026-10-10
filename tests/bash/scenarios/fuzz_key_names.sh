#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
python3 - "$AMAQUET_CLI" "$AMAQUET_URI" <<'PY'
import json,subprocess,sys
cli,uri=sys.argv[1:3]
keys=["space key","colon:key:1","slash/key","dot.key","emoji:🧠:🚀","日本語キー","quote-\"-key","backslash-\\-key","x"*4096]
for i,k in enumerate(keys):
    args=json.dumps({"key":k,"type":"utf8_string","value":f"value-{i}"},ensure_ascii=False)
    subprocess.run([cli,"-uri",uri,"SET",args],check=True,stdout=subprocess.DEVNULL)
    out=subprocess.check_output([cli,"-uri",uri,"GET",json.dumps({"key":k},ensure_ascii=False)],text=True)
    obj=json.loads(out)
    assert obj["value"]==f"value-{i}", (k,obj)
    subprocess.run([cli,"-uri",uri,"DEL",json.dumps({"keys":[k]},ensure_ascii=False)],check=True,stdout=subprocess.DEVNULL)
PY

server_healthy
echo "PASS key names"
