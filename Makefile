APP_NAME    := gmake
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE        := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
BUILD_DIR   := build

# Default target
.PHONY: all
all: clean test build

# ─── Build ───────────────────────────────────────────────────────────
.PHONY: build build-linux build-darwin build-windows

build-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64       ./cmd/gmake
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64       ./cmd/gmake

build-darwin:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64      ./cmd/gmake
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64      ./cmd/gmake

build-windows:
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/gmake
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe ./cmd/gmake

build: build-linux build-darwin build-windows

# ─── Test ────────────────────────────────────────────────────────────
.PHONY: test test-race test-verbose

test:
	go test ./... -count=1

test-race:
	go test ./... -race -count=1

test-verbose:
	go test ./... -v -count=1

# ─── Lint ────────────────────────────────────────────────────────────
.PHONY: lint lint-fix

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run ./... --fix

# ─── SAST / Security ─────────────────────────────────────────────────
.PHONY: sast gosec govulncheck

sast: gosec govulncheck

gosec:
	gosec -no-fail -fmt sarif -out $(BUILD_DIR)/gosec.sarif ./...
	gosec -no-fail ./...

govulncheck:
	govulncheck ./...

# ─── Checks (full CI) ────────────────────────────────────────────────
.PHONY: check

check: lint test-race sast build

# ─── Clean ───────────────────────────────────────────────────────────
.PHONY: clean

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)

# ─── Tidy / Mod ──────────────────────────────────────────────────────
.PHONY: tidy

tidy:
	go mod tidy
	go mod verify
