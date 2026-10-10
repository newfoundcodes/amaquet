#!/usr/bin/env bash
set -Eeuo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$HERE/lib/scenario.sh"

scenario_init

bf="$(scenario_key bloom)"
cb="$(scenario_key cbloom)"
cf="$(scenario_key cuckoo)"
cms="$(scenario_key cms)"
hll="$(scenario_key hll)"
top="$(scenario_key top)"
td="$(scenario_key td)"

create "$bf" bloom_filter '{"expected":10000,"false_positive_rate":0.01}'
create "$cb" counting_bloom_filter '{"expected":10000,"false_positive_rate":0.01}'
create "$cf" cuckoo_filter '{"capacity":4096}'
create "$cms" count_min_sketch '{"width":1024,"depth":5}'
create "$hll" hyperloglog '{"precision":12}'
create "$top" top_k '{"k":5}'
create "$td" t_digest '{"compression":100}'

for ((i = 0; i < AMAQUET_SCENARIO_ITERATIONS; i++)); do
    s="item-$i"

    op "$bf" ADD "{\"value\":\"$s\"}" >/dev/null
    op "$cb" ADD "{\"value\":\"$s\"}" >/dev/null
    op "$cf" ADD "{\"value\":\"$s\"}" >/dev/null
    op "$cms" ADD "{\"value\":\"group$((i % 3))\",\"count\":1}" >/dev/null
    op "$hll" ADD "{\"value\":\"$s\"}" >/dev/null
    op "$top" ADD "{\"value\":\"group$((i % 3))\",\"count\":1}" >/dev/null
    op "$td" ADD "{\"value\":$i}" >/dev/null
done

for i in 0 $((AMAQUET_SCENARIO_ITERATIONS / 2)); do
    assert_true "$(op "$bf" CONTAINS "{\"value\":\"item-$i\"}" | json_extract x)"
    assert_true "$(op "$cb" CONTAINS "{\"value\":\"item-$i\"}" | json_extract x)"
    assert_true "$(op "$cf" CONTAINS "{\"value\":\"item-$i\"}" | json_extract x)"
done

est="$(op "$cms" ESTIMATE '{"value":"group0"}' | json_extract x)"
((est > 0)) || die "CMS estimate must be positive"

hcount="$(op "$hll" COUNT '{}' | json_extract x)"
((hcount > 0)) || die "HLL estimate must be positive"

assert_nonempty "$(op "$top" ITEMS '{}' | json_len)"
assert_nonempty "$(op "$td" QUANTILE '{"q":0.5}' | json_extract x)"

server_healthy
echo "PASS probabilistic workloads"
