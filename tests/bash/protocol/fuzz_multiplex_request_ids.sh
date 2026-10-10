#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
python3 - "$AMAQUET_URI" "$AMAQUET_SCENARIO_ITERATIONS" <<'PY'
import json,socket,struct,sys,urllib.parse
u=urllib.parse.urlparse(sys.argv[1]); host=u.hostname; port=u.port or 13378; n=max(10,int(sys.argv[2])*5)
def frame(i):
    p=json.dumps({"command":"PING","args":{}}).encode(); h=b'AMQT'+bytes([1,1])+struct.pack('>H',0)+struct.pack('>H',1)+b'\x00\x00'+struct.pack('>Q',i)+struct.pack('>I',len(p)); return h+p
s=socket.create_connection((host,port),3); s.settimeout(5)
for i in range(1,n+1): s.sendall(frame(i))
seen=set()
for _ in range(n):
    h=b''
    while len(h)<24:
        x=s.recv(24-len(h)); assert x; h+=x
    assert h[:4]==b'AMQT' and h[4]==1 and h[5]==2
    rid=struct.unpack('>Q',h[12:20])[0]; ln=struct.unpack('>I',h[20:24])[0]
    p=b''
    while len(p)<ln: p+=s.recv(ln-len(p))
    r=json.loads(p); assert r['ok'] and r['result']=='PONG'; seen.add(rid)
assert seen==set(range(1,n+1)), (len(seen),n)
s.close()
PY

server_healthy
echo "PASS multiplex request IDs"
