GO ?= go
NPM ?= npm

.PHONY: test test-backend test-frontend build build-backend build-frontend demo run clean

test: test-backend test-frontend

test-backend:
	cd backend && CGO_ENABLED=1 $(GO) test ./...

test-frontend:
	cd frontend && $(NPM) run test && $(NPM) run typecheck

build: build-backend build-frontend

build-backend:
	mkdir -p backend/bin
	cd backend && CGO_ENABLED=1 $(GO) build -o bin/code-hotspot-analyzer ./cmd/server
	cd backend && CGO_ENABLED=1 $(GO) build -o bin/demo-fixture ./cmd/demo-fixture

build-frontend:
	cd frontend && STATIC_EXPORT=1 $(NPM) run build

demo: build-backend
	./backend/bin/demo-fixture --path ./demo-repository

run:
	@echo "Set ANALYZER_REPO_ROOT and ANALYZER_API_TOKEN first; run make build then export ANALYZER_WEB_DIR=./frontend/out"
	./backend/bin/code-hotspot-analyzer

clean:
	rm -rf backend/bin frontend/out frontend/.next .data demo-repository
