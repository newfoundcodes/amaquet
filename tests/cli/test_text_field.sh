#!/usr/bin/env bash
set -Eeuo pipefail

LIB="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

source "$LIB"
KEY="cli:text_field:$$:$RANDOM"

start_server

cli CREATE "{\"key\":\"$KEY\",\"type\":\"text_field\",\"options\":{\"name\":\"body\",\"value\":\"hello amaquet\",\"weight\":1.5}}" >/dev/null
key_lifecycle "$KEY" text_field

cli OP "{\"key\":\"$KEY\",\"operation\":\"SET\",\"args\":{\"name\":\"body\",\"value\":\"updated text\",\"weight\":2}}" >/dev/null

out="$(cli OP "{\"key\":\"$KEY\",\"operation\":\"GET\",\"args\":{}}")"
assert_contains "$out" updated "GET"

key_del "$KEY"
finish text_field
