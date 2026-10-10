#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
python3 - "$AMAQUET_URI" <<'PY'
import socket,struct,sys,urllib.parse
u=urllib.parse.urlparse(sys.argv[1]); s=socket.create_connection((u.hostname,u.port or 13378),3); s.settimeout(2)
h=b'AMQT'+bytes([1,1])+struct.pack('>H',0)+struct.pack('>H',1)+b'\0\0'+struct.pack('>Q',1)+struct.pack('>I',9*1024*1024)
s.sendall(h)
try: data=s.recv(1); assert data==b''
except (ConnectionResetError,TimeoutError,socket.timeout): pass
s.close()
PY

server_healthy
echo "PASS oversized declared payload rejection"
