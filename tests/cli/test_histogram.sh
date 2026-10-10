#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:histogram:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"histogram\",\"options\":{\"bounds\":[1,5,10,50,100]}}" >/dev/null
key_lifecycle "$KEY" histogram

for v in 1 3 7 45 200; do
    cli OP "{\"key\":\"$KEY\",\"operation\":\"OBSERVE\",\"args\":{\"value\":$v}}" >/dev/null
done

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"SNAPSHOT\",\"args\":{}}")"
assert_contains "$out" count "SNAPSHOT"

key_del "$KEY"
finish histogram
