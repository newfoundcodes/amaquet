#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/custom_server.sh"

ensure_bins

tmp="$(mktemp -d -t amaquet-aof-corrupt.XXXXXX)"
port="$(free_port)"
admin_port="$(free_port)"
cfg="$tmp/amaquet.json"
aof="$tmp/db.aof"

cat >"$cfg" <<JSON
{"protocol":{"host":"127.0.0.1","port":$port,"tls":false,"require_auth":false},"admin":{"host":"127.0.0.1","port":$admin_port},"security":{"bootstrap_token":"aof-corrupt"},"persistence":{"aof_enabled":true,"aof_path":"$aof","fsync":"always"},"limits":{"max_connections":64,"max_payload_bytes":8388608},"compression":{"enabled":true,"algorithm":"auto","min_bytes":1024,"min_savings_percent":8},"data_dir":"$tmp/data"}
JSON
AMAQUET_TEST_TMP="$tmp"

custom_start_server "$cfg" "$port" "$admin_port"
setv corrupt:key integer 123

custom_stop_server
python3 - "$aof" <<'PY'
import os,sys
p=sys.argv[1]; b=bytearray(open(p,'rb').read()); assert len(b)>24
# Flip one byte in record payload, away from the fixed header.
pos=min(len(b)-1,20); b[pos]^=0x55; open(p,'wb').write(b)
PY

set +e
"$AMAQUET_BIN" -config "$cfg" >"$tmp/corrupt.log" 2>&1 &
pid=$!

for _ in {1..40}; do
    if ! kill -0 "$pid" 2>/dev/null; then break; fi
    sleep .05
done

if kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    wait "$pid" 2>/dev/null
    rc=0
else
    wait "$pid"
    rc=$?
fi

set -e
((rc != 0)) || die "corrupt AOF was accepted"

grep -Eqi 'checksum|replay|append-only' "$tmp/corrupt.log" || {
    cat "$tmp/corrupt.log" >&2
    die "corruption failure was not diagnosed"
}

rm -rf "$tmp"
echo "PASS AOF corruption detection"
