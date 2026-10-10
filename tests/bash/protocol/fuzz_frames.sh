#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/common.sh"

start_test_server
: "${AMAQUET_FRAME_FUZZ_ITERATIONS:=250}"

export AMAQUET_FRAME_FUZZ_ITERATIONS AMAQUET_URI
python3 - <<'PY'
import json, os, random, socket, struct
from urllib.parse import urlparse
u=urlparse(os.environ['AMAQUET_URI']); host=u.hostname; port=u.port or 13378
iterations=int(os.environ.get('AMAQUET_FRAME_FUZZ_ITERATIONS','250'))

def frame(magic=b'AMQT', version=1, kind=1, flags=0, opcode=1, rid=1, payload=b''):
    h=bytearray(24); h[0:4]=magic; h[4]=version; h[5]=kind
    struct.pack_into('>H',h,6,flags); struct.pack_into('>H',h,8,opcode)
    struct.pack_into('>Q',h,12,rid); struct.pack_into('>I',h,20,len(payload)); return bytes(h)+payload

def send(data, read=False):
    try:
        with socket.create_connection((host,port),0.5) as s:
            s.settimeout(0.25); s.sendall(data)
            if read:
                try: s.recv(4096)
                except (socket.timeout,ConnectionError,OSError): pass
    except (ConnectionError,OSError):
        pass

fixed=[
 b'', b'N', b'AMQT\x01', b'XXXX'+bytes(20),
 frame(version=0,payload=b'{}'), frame(version=255,payload=b'{}'),
 frame(kind=0,payload=b'{}'), frame(kind=255,payload=b'{}'),
 frame(opcode=0,payload=b'{}'), frame(opcode=65535,payload=b'{}'),
 frame(payload=b'{'), frame(payload=b'not-json'),
 frame(payload=json.dumps({'command':'NO_SUCH_COMMAND','args':{}}).encode()),
]

# Header declaring a payload larger than the configured test limit, without sending it.
h=bytearray(frame(payload=b'')[:24]); struct.pack_into('>I',h,20,16*1024*1024); fixed.append(bytes(h))
for x in fixed: send(x, True)

for i in range(iterations):
    choice=random.randrange(7)
    if choice==0:
        data=os.urandom(random.randint(1,80))
    elif choice==1:
        p=os.urandom(random.randint(0,512)); data=frame(version=random.randrange(256),payload=p)
    elif choice==2:
        p=os.urandom(random.randint(0,512)); data=frame(kind=random.randrange(256),payload=p)
    elif choice==3:
        p=os.urandom(random.randint(0,512)); data=frame(opcode=random.randrange(65536),payload=p)
    elif choice==4:
        p=json.dumps({'command':random.choice(['PING','GET','SET','CREATE','OP','X'*random.randint(1,100)]),'args':{'noise':random.randint(-2**31,2**31-1)}}).encode(); data=frame(rid=random.getrandbits(64),payload=p)
    elif choice==5:
        data=bytearray(frame(payload=os.urandom(random.randint(0,100)))); data[random.randrange(len(data))]^=random.randrange(1,256); data=bytes(data)
    else:
        full=frame(payload=os.urandom(random.randint(1,512))); data=full[:random.randint(1,len(full)-1)]
    send(data, i%20==0)
print(f'raw-frame fuzz cases sent: {len(fixed)+iterations}')
PY

# A healthy PING after malformed frames proves the server remained available.
amaquet PING '{}' >/dev/null
log "FRAME FUZZ PASS ($AMAQUET_FRAME_FUZZ_ITERATIONS randomized cases)"
