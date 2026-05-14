.PHONY: build build-token-savior build-ts-compat build-all build-linux clean test test-compat lint help

VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS    = -ldflags="-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

build: build-token-savior

build-token-savior:
	go build $(LDFLAGS) -o bin/token-savior ./cmd/token-savior

build-ts-compat:
	go build $(LDFLAGS) -o bin/ts-compat ./cmd/ts-compat

build-all: build-token-savior build-ts-compat

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/token-savior-linux-amd64 ./cmd/token-savior
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/token-savior-linux-arm64 ./cmd/token-savior

clean:
	rm -rf bin/ coverage.out

test:
	@if [ -n "$$(find . -name '*.go' -not -path './vendor/*' 2>/dev/null | head -1)" ]; then \
		go test -race -count=1 ./...; \
	else \
		echo "test: no Go files yet — skipping"; \
	fi

test-compat: build-ts-compat
	./bin/ts-compat -fixture testdata/fixtures/go-small -python token-savior

lint:
	@if [ -n "$$(find . -name '*.go' -not -path './vendor/*' 2>/dev/null | head -1)" ]; then \
		golangci-lint run ./...; \
	else \
		echo "lint: no Go files yet — skipping"; \
	fi

help:
	@echo "Targets:"
	@echo "  build          - Build token-savior"
	@echo "  build-all      - Build token-savior + ts-compat"
	@echo "  build-linux    - Static Linux amd64/arm64 builds"
	@echo "  test           - Run unit tests"
	@echo "  test-compat    - Run compat harness against Python v3"
	@echo "  lint           - golangci-lint"
	@echo "  clean          - Remove build artefacts"
