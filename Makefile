BINARY := ares
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO_PACKAGES := $(shell go list ./... | grep -v '/website/')
GO_PACKAGE_DIRS := $(shell go list -f '{{.Dir}}' ./... | grep -v '/website/')

.PHONY: build test smoke integration release-check lint actionlint install clean vet markdown website-install website-typecheck website-build website-ci ci

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test -race -coverprofile=coverage.out $(GO_PACKAGES)
	@go tool cover -func=coverage.out | tail -1

markdown:
	bunx markdownlint-cli2

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
	go vet $(GO_PACKAGES)

lint:
	golangci-lint run $(GO_PACKAGE_DIRS)

actionlint:
	actionlint

website-install:
	cd website && bun install --frozen-lockfile

website-typecheck:
	cd website && bunx tsc --noEmit

website-build:
	cd website && bun run build

website-ci: website-install website-typecheck website-build

ci: markdown test vet lint actionlint build smoke integration release-check website-ci

clean:
	rm -f $(BINARY) coverage.out

cover: test
	go tool cover -html=coverage.out

.DEFAULT_GOAL := build
