#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
python3 - "$AMAQUET_URI" "$AMAQUET_FUZZ_SEED" <<'PY'
import json,random,socket,struct,sys,time,urllib.parse
u=urllib.parse.urlparse(sys.argv[1]); r=random.Random(int(sys.argv[2])); p=json.dumps({"command":"PING","args":{}}).encode(); b=b'AMQT'+bytes([1,1])+struct.pack('>H',0)+struct.pack('>H',1)+b'\0\0'+struct.pack('>Q',12345)+struct.pack('>I',len(p))+p
s=socket.create_connection((u.hostname,u.port or 13378),3); s.settimeout(5); i=0
while i<len(b):
    n=r.randint(1,min(7,len(b)-i)); s.sendall(b[i:i+n]); i+=n; time.sleep(0.001)
h=s.recv(24); assert len(h)==24 and struct.unpack('>Q',h[12:20])[0]==12345; ln=struct.unpack('>I',h[20:24])[0]; out=b''
while len(out)<ln: out+=s.recv(ln-len(out))
assert json.loads(out)['result']=='PONG'; s.close()
PY

server_healthy
echo "PASS partial frame writes"
