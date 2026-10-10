#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    python3 - "$AMAQUET_URI" "$i" <<'PY'
import socket,sys,urllib.parse,os
u=urllib.parse.urlparse(sys.argv[1]); s=socket.create_connection((u.hostname,u.port or 13378),2); s.sendall(os.urandom(4+int(sys.argv[2])%40)); s.close()
PY
    amaquet PING '{}' >/dev/null
done

server_healthy
echo "PASS reconnect after malformed clients"
