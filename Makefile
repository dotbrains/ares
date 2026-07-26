BINARY := ares
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test smoke integration release-check lint install clean vet

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

smoke: build
	tests/smoke.sh

integration:
	tests/integration.sh

release-check:
	@if git remote get-url origin >/dev/null 2>&1; then \
		goreleaser check; \
	else \
		echo "skipping goreleaser check: git remote 'origin' is not configured"; \
	fi

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -f $(BINARY) coverage.out

cover: test
	go tool cover -html=coverage.out

.DEFAULT_GOAL := build
