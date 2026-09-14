.PHONY: all build frontend backend clean release \
	lint check check-standards golangci test test-integration ensure-dist \
	build-darwin-amd64 build-darwin-arm64 \
	build-linux-amd64 build-linux-arm64 \
	build-windows-amd64 build-windows-arm64

BIN_DIR := bin
BINARY  := nexa
PKG     := ./cmd
CGO     := 0

# Release version, injected into core/version at build time.
# Defaults to a git tag (or "dev" when none is present); override with
# `make release VERSION=v1.2.3` (CI passes the release tag).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

LDFLAGS := -s -w -X github.com/teexue/nexa/core/version.Version=$(VERSION)

all: build

# Push CI runs this target. It must include every check that workflow runs.
build: check test golangci frontend backend

# go:embed all:dist in cmd/static.go requires the directory to exist and
# contain at least one file. Frontend build replaces this with real assets.
ensure-dist:
	@mkdir -p cmd/dist
	@touch cmd/dist/.keep

lint: ensure-dist
	go vet ./...
	pnpm --dir frontend exec eslint src --max-warnings 0
	pnpm --dir frontend run format:check

check-standards:
	python3 scripts/check-standards.py --quick

check: lint check-standards

# Same linter and config as .github/workflows/ci.yml. Not part of make lint.
golangci: ensure-dist
	golangci-lint run

test: ensure-dist
	go test ./...
	pnpm --dir frontend test

test-integration: ensure-dist
	go test -tags=integration ./test/integration/...

frontend:
	pnpm --dir frontend run build

backend: ensure-dist
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

clean:
	rm -rf $(BIN_DIR)/ cmd/dist

build-darwin-amd64 build-darwin-arm64 \
build-linux-amd64 build-linux-arm64 \
build-windows-amd64 build-windows-arm64: ensure-dist

define GO_CROSS
	CGO_ENABLED=$(CGO) GOOS=$(1) GOARCH=$(2) \
		go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-$(1)-$(2)$(3) $(PKG)
endef

build-darwin-amd64:
	$(call GO_CROSS,darwin,amd64,)

build-darwin-arm64:
	$(call GO_CROSS,darwin,arm64,)

build-linux-amd64:
	$(call GO_CROSS,linux,amd64,)

build-linux-arm64:
	$(call GO_CROSS,linux,arm64,)

build-windows-amd64:
	$(call GO_CROSS,windows,amd64,.exe)

build-windows-arm64:
	$(call GO_CROSS,windows,arm64,.exe)

# ── Cross-compile all platforms (frontend once) ───────────────────
# Usage: make release VERSION=v1.2.3

release: check frontend \
	build-darwin-amd64 build-darwin-arm64 \
	build-linux-amd64 build-linux-arm64 \
	build-windows-amd64 build-windows-arm64
	@echo ""
	@echo "Built version: $(VERSION)"
	@echo "Artifacts in $(BIN_DIR)/:"
	@ls -lh $(BIN_DIR)/$(BINARY)-*
