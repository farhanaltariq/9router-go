BINARY_NAME := 9router-go
# Central version — single source: VERSION file, fallback to version.json, then git
VERSION ?= $(shell cat VERSION 2>/dev/null || (cat version.json 2>/dev/null | grep -o '"latestVersion": *"[^"]*"' | cut -d'"' -f4) || git describe --tags --always 2>/dev/null || echo "1.0.0")
PORT ?= 20130
DATA_DIR ?= $(HOME)/.9router
RTK ?=
CAVEMAN ?=
PONYTAIL ?=
AUTO_UPDATE ?= false

LDFLAGS := -s -w -X '9router/proxy/internal/updater.CurrentVersion=$(VERSION)'

.PHONY: build run dev version update test test-short vet bench bench-go cross docker docker-build clean help web-build

## web-build — build frontend static assets (Svelte 5/Vite 8) into backend/web/dist
web-build:
	@if [ ! -f backend/web/dist/index.html ] || [ "$$FORCE" = "1" ]; then \
		echo "Building web SPA assets from frontend/..."; \
		cd frontend && bun install --frozen-lockfile && bun run build; \
	fi

## build — compile binary with embedded version & SPA assets
build: web-build
	cd backend && go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME) ./cmd/9router-go/

## run — start proxy (PORT=20130)
run: build
	PORT=$(PORT) DATA_DIR=$(DATA_DIR) ./$(BINARY_NAME) $(if $(RTK),--rtk=$(RTK)) $(if $(CAVEMAN),--caveman=$(CAVEMAN)) $(if $(PONYTAIL),--ponytail=$(PONYTAIL)) --auto-update=$(AUTO_UPDATE)

## dev — start with go run (auto-rebuild)
dev:
	PORT=$(PORT) DATA_DIR=$(DATA_DIR) go run -ldflags="$(LDFLAGS)" ./backend/cmd/9router-go/ $(if $(RTK),--rtk=$(RTK)) $(if $(CAVEMAN),--caveman=$(CAVEMAN)) $(if $(PONYTAIL),--ponytail=$(PONYTAIL)) --auto-update=$(AUTO_UPDATE)

## version — display binary version info
version: build
	./$(BINARY_NAME) version

## update — check and install binary self-update
update: build
	./$(BINARY_NAME) update

## test — run all backend unit tests
test:
	cd backend && go test -short ./... -v

## test-short — run backend tests (quiet)
test-short:
	cd backend && go test -short ./...

## test-live — run live upstream integration tests
test-live:
	cd backend && go test ./... -v

## vet — run go vet static analysis
vet:
	cd backend && go vet ./...

## bench — run bash comparison benchmark
bench: build
	bash benchmark/run_comparison.sh

## bench-go — run native Go high-throughput benchmark
bench-go:
	cd backend && go run ../benchmark/runner.go

## cross — cross-compile Linux/macOS/Windows release binaries
cross: web-build
	cd backend && GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-linux-amd64 ./cmd/9router-go/
	cd backend && GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-linux-arm64 ./cmd/9router-go/
	cd backend && GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-darwin-amd64 ./cmd/9router-go/
	cd backend && GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-darwin-arm64 ./cmd/9router-go/
	cd backend && GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ../$(BINARY_NAME)-windows-amd64.exe ./cmd/9router-go/
	@ls -lh $(BINARY_NAME)-*
	@(sha256sum $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64 $(BINARY_NAME)-windows-amd64.exe 2>/dev/null || shasum -a 256 $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-darwin-amd64 $(BINARY_NAME)-darwin-arm64 $(BINARY_NAME)-windows-amd64.exe) > SHA256SUMS.txt
	@cat SHA256SUMS.txt

## docker — docker compose up
docker:
	docker compose up -d

## docker-build — build Docker image only
docker-build:
	docker build -t $(BINARY_NAME) .

## clean — remove build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf backend/web/dist frontend/dist

## help — show targets
help:
	@echo "9router-go — Makefile targets:"
	@grep -E '^## ' Makefile | sed 's/## /  make /' | sed 's/ — /  /'
	@echo ""
	@echo "Options:"
	@echo "  make run PORT=3000 VERSION=1.1.0"
	@echo "  make run DATA_DIR=/path/to/data"
	@echo "  make run CAVEMAN=true PONYTAIL=true AUTO_UPDATE=true"
	@echo "  make run RTK=false"
