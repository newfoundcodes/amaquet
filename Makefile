.PHONY: all build go-build test race vet vulncheck test-bash fuzz-smoke fuzz-types fuzz-protocol fuzz-scenarios fuzz-isolated fuzz-developer fuzz-go fuzz-all chaos chaos-smoke benchmark docs-check docs-install docs-build docs-serve run init keygen restore clean docker

DOCS_INSTALL_STAMP := docs/node_modules/.amaquet-install-stamp
GOVULNCHECK ?= go run golang.org/x/vuln/cmd/govulncheck@latest

all: test build

build: go-build

go-build:
	mkdir -p bin
	go build -trimpath -o bin/amaquet ./cmd/amaquet
	go build -trimpath -o bin/amaquet-cli ./cmd/amaquet-cli
	go build -trimpath -o bin/amaquet-keygen ./cmd/amaquet-keygen
	go build -trimpath -o bin/amaquet-restore ./cmd/amaquet-restore

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

vulncheck:
	$(GOVULNCHECK) ./...

test-bash:
	./tests/bash/run_all.sh

fuzz-smoke:
	AMAQUET_FUZZ_ITERATIONS=2 ./tests/bash/run_fuzz.sh

fuzz-types:
	./tests/bash/run_fuzz.sh
	./tests/cli/run_all.sh

fuzz-protocol:
	AMAQUET_FRAME_FUZZ_ITERATIONS="$${AMAQUET_FRAME_FUZZ_ITERATIONS:-100}" ./tests/bash/run_protocol_fuzz.sh

fuzz-scenarios:
	AMAQUET_SCENARIO_ITERATIONS="$${AMAQUET_SCENARIO_ITERATIONS:-10}" ./tests/bash/run_scenarios.sh

fuzz-isolated:
	./tests/bash/run_isolated_fuzz.sh

fuzz-developer:
	./tests/bash/run_developer_fuzz.sh

fuzz-go:
	./scripts/run_go_fuzz.sh

# Runs every fuzz layer once: per-type, scenarios, isolated, protocol, and native Go fuzzing.
fuzz-all: fuzz-types fuzz-scenarios fuzz-isolated fuzz-protocol fuzz-go

chaos:
	./tests/bash/run_chaos.sh

chaos-smoke:
	AMAQUET_SOAK_SECONDS=2 AMAQUET_SOAK_KEYS=16 AMAQUET_CHURN_CONNECTIONS=24 AMAQUET_CHURN_PARALLEL=4 AMAQUET_TTL_CHURN=50 AMAQUET_RESTART_ITERATIONS=2 AMAQUET_MEMORY_WRITES=80 ./tests/bash/run_chaos.sh

benchmark:
	./scripts/run_benchmarks.sh

docs-check:
	python3 docs/scripts/check_docs.py

$(DOCS_INSTALL_STAMP): docs/package.json docs/package-lock.json
	cd docs && npm ci --no-audit --no-fund
	touch $@

docs-install: $(DOCS_INSTALL_STAMP)

docs-build: docs-check docs-install
	cd docs && npm run build

docs-serve: docs-check docs-install
	cd docs && npm run dev

run:
	go run ./cmd/amaquet --config ./amaquet.json

init:
	go run ./cmd/amaquet --init-config --config ./amaquet.json

keygen:
	mkdir -p keys
	go run ./cmd/amaquet-keygen --public ./keys/amaquet-public.pem --private ./keys/amaquet-private.pem

restore:
	go run ./cmd/amaquet-restore --source "$${SOURCE}" --target "$${TARGET}"

clean:
	rm -rf bin docs/dist docs/node_modules docs/.astro site .venv-docs

docker:
	docker build -t amaquet:local .

.PHONY: test-compression
test-compression:
	bash tests/bash/test_compression.sh
