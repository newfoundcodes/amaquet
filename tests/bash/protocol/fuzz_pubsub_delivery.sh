#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
python3 - "$AMAQUET_URI" <<'PY'
import json,socket,struct,sys,urllib.parse
u=urllib.parse.urlparse(sys.argv[1]); addr=(u.hostname,u.port or 13378)
def enc(kind,op,rid,obj):
    p=json.dumps(obj,separators=(',',':')).encode(); return b'AMQT'+bytes([1,kind])+struct.pack('>H',0)+struct.pack('>H',op)+b'\0\0'+struct.pack('>Q',rid)+struct.pack('>I',len(p))+p
def read(s):
    h=b''
    while len(h)<24:
        x=s.recv(24-len(h)); assert x; h+=x
    n=struct.unpack('>I',h[20:24])[0]; p=b''
    while len(p)<n: p+=s.recv(n-len(p))
    return h[5],struct.unpack('>H',h[8:10])[0],struct.unpack('>Q',h[12:20])[0],json.loads(p)
sub=socket.create_connection(addr,3); pub=socket.create_connection(addr,3); sub.settimeout(5); pub.settimeout(5)
sub.sendall(enc(1,1,1,{"command":"CREATE","args":{"key":"ps:fuzz","type":"pubsub","options":{}}})); assert read(sub)[3]['ok']
sub.sendall(enc(1,4,77,{"key":"ps:fuzz","channel":"events","buffer":8})); k,op,rid,resp=read(sub); assert resp['ok'] and rid==77
pub.sendall(enc(1,1,1,{"command":"OP","args":{"key":"ps:fuzz","operation":"PUBLISH","args":{"channel":"events","value":{"type":"utf8_string","value":"hello"}}}})); assert read(pub)[3]['result']==1
kind,op,rid,event=read(sub); assert kind==3 and op==4 and rid==77; assert event['channel']=='events'; assert event['payload']['value']=='hello'
sub.sendall(enc(1,5,78,{"key":"ps:fuzz","channel":"events"})); assert read(sub)[3]['ok']; sub.close(); pub.close()
PY

server_healthy
echo "PASS Pub/Sub subscribe-deliver-unsubscribe"
