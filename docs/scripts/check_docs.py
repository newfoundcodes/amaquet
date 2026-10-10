#!/usr/bin/env python3

from pathlib import Path
import json
import re
import sys

root=Path(__file__).resolve().parents[2]
docs=root/'docs'
errors=[]

markdown_pages=sorted(
    p for p in docs.rglob('*.md')
    if 'node_modules' not in p.parts and '.astro' not in p.parts
)

pat=re.compile(r'\[[^\]]*\]\(([^)]+)\)')
for p in markdown_pages:
    text=p.read_text(encoding='utf-8')
    for target in pat.findall(text):
        target=target.split('#',1)[0].strip()

        if not target or '://' in target or target.startswith(('mailto:', '/')):
            continue

        q=(p.parent/target).resolve()
        if docs.resolve() not in q.parents and q != docs.resolve():
            errors.append(f'{p.relative_to(root)}: link escapes docs: {target}')
        elif not q.exists():
            errors.append(f'{p.relative_to(root)}: missing link target: {target}')

manifest=json.loads((root/'tests/bash/type-manifest.json').read_text())
names=[x['name'] for x in manifest]

if len(names)!=91 or len(set(names))!=91:
    errors.append(f'type manifest expected 91 unique entries, got {len(names)}/{len(set(names))}')

datatype_source=(root/'internal/core/datatype.go').read_text(encoding='utf-8')
registry_block=re.search(r'var typeNames = map\[DataType\]string\{(.*?)\n\}', datatype_source, re.S)
registry_names=set(re.findall(r'Type[A-Za-z0-9]+:\s*"([^"]+)"', registry_block.group(1) if registry_block else ''))

if registry_names != set(names):
    errors.append(
        'type manifest differs from core registry: '
        f'missing={sorted(registry_names-set(names))}, extra={sorted(set(names)-registry_names)}'
    )

operations_page=(docs/'data-types/operations.md').read_text(encoding='utf-8')
catalog_page=(docs/'data-types/index.md').read_text(encoding='utf-8')
catalog_rows={}
for line in catalog_page.splitlines():
    cells=[cell.strip() for cell in line.strip().strip('|').split('|')]
    if len(cells) != 4:
        continue
    match=re.search(r'\[`([^`]+)`\]', cells[0])
    if match and cells[3].isdigit():
        catalog_rows[match.group(1)]=(cells[2], int(cells[3]))

for n in names:
    for rel in [f'tests/bash/types/test_{n}.sh', f'tests/bash/fuzz/fuzz_{n}.sh', f'docs/data-types/types/{n}.md']:
        if not (root/rel).exists(): errors.append(f'missing generated artifact: {rel}')

    page_path=docs/f'data-types/types/{n}.md'
    if page_path.exists():
        page_text=page_path.read_text(encoding='utf-8')
        entry=next(x for x in manifest if x['name'] == n)

        if f'**Wire type:** `{n}`' not in page_text:
            errors.append(f'{page_path.relative_to(root)}: wire type does not match manifest')

        if f'**Construction:** `{entry["construction"]}`' not in page_text:
            errors.append(f'{page_path.relative_to(root)}: construction does not match manifest')

        for op in entry['operations']:
            if f'`{op}`' not in page_text:
                errors.append(f'{page_path.relative_to(root)}: missing manifest operation {op}')

        op_headings=set(re.findall(r'^#### `OP ([A-Z_]+)`', page_text, re.M))
        for op in entry['operations']:
            if op not in op_headings and not (entry['construction'] == 'SET' and op == 'GET'):
                errors.append(
                    f'{page_path.relative_to(root)}: missing operation payload section {op}'
                )

    if f'`{n}`' not in operations_page:
        errors.append(f'docs/data-types/operations.md: missing registered type {n}')

    entry=next(x for x in manifest if x['name'] == n)
    if n not in catalog_rows:
        errors.append(f'docs/data-types/index.md: missing catalog row for {n}')
    else:
        construction,operation_count=catalog_rows[n]
        if not construction.startswith(entry['construction']):
            errors.append(f'docs/data-types/index.md: construction for {n} differs from manifest')
        if operation_count != len(entry['operations']):
            errors.append(
                f'docs/data-types/index.md: operation count for {n} is {operation_count}, '
                f'want {len(entry["operations"])}'
            )

dispatcher_source=(root/'internal/server/dispatcher.go').read_text(encoding='utf-8')
execute_match=re.search(r'func \(d \*Dispatcher\) execute\(.*?\n\}(?=\n\nfunc \(d \*Dispatcher\) cas)', dispatcher_source, re.S)
commands=set()

if execute_match:
    for case_values in re.findall(r'case\s+((?:"[A-Z_]+"\s*,?\s*)+):', execute_match.group(0)):
        commands.update(re.findall(r'"([A-Z_]+)"', case_values))
else:
    errors.append('internal/server/dispatcher.go: cannot derive command inventory')

command_docs=(docs/'protocol/commands.md').read_text(encoding='utf-8')
for command in sorted(commands):
    if f'`{command}`' not in command_docs:
        errors.append(f'docs/protocol/commands.md: missing dispatcher command {command}')

frame_source=(root/'internal/protocol/frame.go').read_text(encoding='utf-8')
opcode_block=re.search(r'const \(\n\s*OpCommand.*?\n\)', frame_source, re.S)
opcode_names=re.findall(r'^\s*Op([A-Za-z0-9]+)\s+Opcode(?:\s*=.*)?$', opcode_block.group(0) if opcode_block else '', re.M)

opcodes=set(opcode_names)
framing_docs=(docs/'protocol/framing.md').read_text(encoding='utf-8')

for opcode in sorted(opcodes):
    label=re.sub(r'(?<!^)(?=[A-Z])', '_', opcode).upper()
    if f'`{label}`' not in framing_docs:
        errors.append(f'docs/protocol/framing.md: missing opcode {label}')

config_source=(root/'internal/config/config.go').read_text(encoding='utf-8')
config_docs=(docs/'reference/configuration.md').read_text(encoding='utf-8')

prefix_by_struct={
    'ProtocolConfig':'protocol', 'AdminConfig':'admin', 'SecurityConfig':'security',
    'PersistenceConfig':'persistence', 'LimitsConfig':'limits',
    'CompressionConfig':'compression',
}

config_paths=set()
for struct_name,prefix in prefix_by_struct.items():
    match=re.search(rf'type {struct_name} struct \{{(.*?)\n\}}', config_source, re.S)

    if not match:
        errors.append(f'internal/config/config.go: cannot derive {struct_name} fields')
        continue

    for field in re.findall(r'json:"([^",]+)', match.group(1)):
        config_paths.add(f'{prefix}.{field}')

config_paths.add('data_dir')
for path in sorted(config_paths):
    if f'`{path}`' not in config_docs:
        errors.append(f'docs/reference/configuration.md: missing config field {path}')

env_names=set(re.findall(r'os\.Getenv\("(AMAQUET_[A-Z0-9_]+)"\)', config_source))
for name in sorted(env_names):
    if f'`{name}`' not in config_docs:
        errors.append(f'docs/reference/configuration.md: missing environment override {name}')

admin_source=(root/'internal/admin/http.go').read_text(encoding='utf-8')
admin_routes=set(re.findall(r'mux\.HandleFunc\("([^"]+)"', admin_source))
admin_docs=(docs/'reference/admin-api.md').read_text(encoding='utf-8')

for route in sorted(admin_routes):
    if f'`{route}`' not in admin_docs and route not in admin_docs:
        errors.append(f'docs/reference/admin-api.md: missing admin route {route}')

openapi_path=root/'internal/admin/openapi.json'
try:
    openapi=json.loads(openapi_path.read_text(encoding='utf-8'))
except (OSError, json.JSONDecodeError) as exc:
    errors.append(f'internal/admin/openapi.json: invalid or missing OpenAPI document: {exc}')
    openapi={}

if not str(openapi.get('openapi', '')).startswith('3.1.'):
    errors.append('internal/admin/openapi.json: expected OpenAPI 3.1.x')

openapi_paths=set(openapi.get('paths', {}))
route_paths={route for route in admin_routes if not route.endswith('/')}
route_paths.update({
    '/api/members/{id}', '/api/members/{id}/invite', '/api/api-keys/{id}',
})

for route in sorted(route_paths):
    if route not in openapi_paths:
        errors.append(f'internal/admin/openapi.json: missing registered route {route}')

operation_ids=[]
for item in openapi.get('paths', {}).values():
    for method,operation in item.items():
        if method.lower() in {'get','post','put','patch','delete','head','options','trace'} and isinstance(operation, dict):
            operation_id=operation.get('operationId')
            if not operation_id:
                errors.append(f'internal/admin/openapi.json: {method.upper()} operation missing operationId')
            else:
                operation_ids.append(operation_id)

if len(operation_ids) != len(set(operation_ids)):
    errors.append('internal/admin/openapi.json: duplicate operationId values')

client_source=(root/'pkg/amaquet/client.go').read_text(encoding='utf-8')
client_exports=set(re.findall(r'^func (?:\([^)]*\) )?([A-Z][A-Za-z0-9_]*)\(', client_source, re.M))
client_docs=(docs/'reference/go-client.md').read_text(encoding='utf-8')

for symbol in sorted(client_exports):
    if not re.search(rf'`[^`]*\b{re.escape(symbol)}\b`', client_docs):
        errors.append(f'docs/reference/go-client.md: missing public client symbol {symbol}')

coverage_docs=(docs/'reference/implementation-coverage.md').read_text(encoding='utf-8')
production_paths=['cmd/amaquet','cmd/amaquet-cli','cmd/amaquet-keygen','cmd/amaquet-restore','pkg/amaquet']
production_paths += [str(p.relative_to(root)) for p in sorted((root/'internal').iterdir()) if p.is_dir()]

for path in production_paths:
    if f'`{path}`' not in coverage_docs and f'`{path}' not in coverage_docs:
        errors.append(f'docs/reference/implementation-coverage.md: missing production component {path}')

scenario_count=len(list((root/'tests/bash/scenarios').glob('fuzz_*.sh')))
security_count=len(list((root/'tests/bash/security').glob('fuzz_*.sh')))
persistence_count=len(list((root/'tests/bash/persistence').glob('fuzz_*.sh')))
protocol_fuzz_count=len(list((root/'tests/bash/protocol').glob('fuzz_*.sh')))

if scenario_count < 28: errors.append(f'expected at least 28 developer scenario fuzzers, got {scenario_count}')
if security_count < 3: errors.append(f'expected at least 3 security fuzzers, got {security_count}')
if persistence_count < 5: errors.append(f'expected at least 5 persistence fuzzers, got {persistence_count}')
if protocol_fuzz_count < 6: errors.append(f'expected at least 6 protocol fuzzers, got {protocol_fuzz_count}')

for rel in ['tests/bash/run_scenarios.sh','tests/bash/run_isolated_fuzz.sh','tests/bash/run_developer_fuzz.sh','scripts/run_go_fuzz.sh','docs/development/developer-fuzzing.md']:
    if not (root/rel).exists(): errors.append(f'missing extended fuzz artifact: {rel}')

go_fuzz_targets=0
for fp in (root/'internal').rglob('*_fuzz_test.go'):
    go_fuzz_targets += len(re.findall(r'^func\s+Fuzz[A-Za-z0-9_]+\(', fp.read_text(), re.M))

if go_fuzz_targets < 8: errors.append(f'expected at least 8 native Go fuzz targets, got {go_fuzz_targets}')

astro_config_path=docs/'astro.config.mjs'
if not astro_config_path.exists():
    errors.append('docs/astro.config.mjs: missing Astro configuration')
else:
    astro_config=astro_config_path.read_text(encoding='utf-8')
    nav_slugs=re.findall(r'''slug:\s*["']([^"']+)["']''', astro_config)

    if not nav_slugs:
        errors.append('docs/astro.config.mjs: sidebar contains no documentation pages')

    for slug in nav_slugs:
        rel=f'{slug}.md'
        if not (docs/rel).exists(): errors.append(f'Astro sidebar missing: {rel}')

for page in markdown_pages:
    if page == docs/'README.md':
        continue

    text=page.read_text(encoding='utf-8')
    frontmatter=re.match(r'^---\n(.*?)\n---\n', text, re.S)

    if not frontmatter or not re.search(r'^title:\s*.+$', frontmatter.group(1), re.M):
        errors.append(f'{page.relative_to(root)}: missing Astro title frontmatter')
    elif re.search(r'^title:\s*["\']?.*`', frontmatter.group(1), re.M):
        errors.append(f'{page.relative_to(root)}: frontmatter titles are plain text; remove Markdown backticks')

    headings=[]
    for line_number,line in enumerate(text.splitlines(), start=1):
        match=re.match(r'^(#{2,4})\s+(.+)$', line)
        if match:
            headings.append((line_number, len(match.group(1)), match.group(2)))

    lines=text.splitlines()
    for index,(line_number,level,title) in enumerate(headings):
        end=len(lines)+1
        for next_line,next_level,_ in headings[index+1:]:
            if next_level <= level:
                end=next_line
                break

        in_fence=False
        has_prose=False
        for line in lines[line_number:end-1]:
            stripped=line.strip()
            if stripped.startswith('```'):
                in_fence=not in_fence
                continue
            if in_fence or not stripped:
                continue
            if re.match(r'^(?:[-*+] |\d+[.)] |\| |>|<)', stripped):
                continue
            if re.match(r'^\[[^]]+\]:', stripped):
                continue
            has_prose=True
            break

        if not has_prose:
            errors.append(f'{page.relative_to(root)}:{line_number}: section {title!r} needs an explanatory paragraph')

if errors:
    print('\n'.join('ERROR: '+e for e in errors)); sys.exit(1)

print(f'docs check passed: {len(markdown_pages)} Markdown pages, {len(names)} types, {scenario_count} developer scenarios, {go_fuzz_targets} Go fuzz targets')
