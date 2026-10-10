#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init
b="$(scenario_key bt)"
h="$(scenario_key hi)"
r="$(scenario_key radix)"
inv="$(scenario_key inv)"
sec="$(scenario_key sec)"

create "$b" btree_index '{"degree":8}'
create "$h" hash_index '{}'
create "$r" radix_tree '{}'
create "$inv" inverted_index '{}'
create "$sec" secondary_index '{}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    val=$(printf '%04d' "$i")
    op "$b" INSERT "{\"value\":\"$val\",\"id\":\"id$i\"}" >/dev/null
    op "$h" ADD "{\"value\":\"group$((i % 4))\",\"id\":\"id$i\"}" >/dev/null
    op "$r" INSERT "{\"value\":\"prefix-$i\",\"id\":\"id$i\"}" >/dev/null
    op "$inv" INDEX "{\"id\":\"id$i\",\"text\":\"amaquet database document $i group$((i % 3))\"}" >/dev/null
    op "$sec" ADD "{\"field\":\"group\",\"value\":\"g$((i % 5))\",\"id\":\"id$i\"}" >/dev/null
done

assert_nonempty "$(op "$b" RANGE '{"value":"0000","max":"9999","limit":1000}' | json_len)"
assert_nonempty "$(op "$h" LOOKUP '{"value":"group0"}' | json_len)"
assert_nonempty "$(op "$r" PREFIX '{"prefix":"prefix-","limit":1000}' | json_len)"
assert_nonempty "$(op "$inv" SEARCH '{"query":"amaquet database","limit":20}' | json_len)"
assert_nonempty "$(op "$sec" LOOKUP '{"field":"group","value":"g0"}' | json_len)"

server_healthy
echo "PASS index/search workloads"
