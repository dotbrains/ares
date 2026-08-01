BINARY := ares
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO_PACKAGES := $(shell go list ./...)
GO_PACKAGE_DIRS := $(shell go list -f '{{.Dir}}' ./...)

.PHONY: build test smoke integration release-artifact-smoke release-check release-preflight lint actionlint budgets security install clean vet markdown ci

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

install:
	go install -ldflags "$(LDFLAGS)" .

test:
	go test -race -coverprofile=coverage.out $(GO_PACKAGES)
	@go tool cover -func=coverage.out | tail -1

markdown:
	markdownlint-cli2

smoke: build
	tests/smoke.sh

release-artifact-smoke: build
	tests/release-artifact-smoke.sh

integration:
	tests/integration.sh

release-check:
	@if git remote get-url origin >/dev/null 2>&1; then \
		goreleaser check; \
	else \
		echo "skipping goreleaser check: git remote 'origin' is not configured"; \
	fi

release-preflight:
	@if git remote get-url origin >/dev/null 2>&1; then \
		tag="$$(git describe --tags --exact-match 2>/dev/null || true)"; \
		if [ -n "$$tag" ]; then \
			git fetch origin main --quiet; \
			git merge-base --is-ancestor "$$tag" origin/main; \
		else \
			echo "skipping release tag ancestry check: HEAD is not tagged"; \
		fi; \
	else \
		echo "skipping release preflight: git remote 'origin' is not configured"; \
	fi

vet:
	go vet $(GO_PACKAGES)

lint:
	golangci-lint run $(GO_PACKAGE_DIRS)

actionlint:
	actionlint

budgets:
	scripts/check-file-sizes.sh
	scripts/check-flat-directories.sh
	scripts/check-provider-docs.sh

security:
	govulncheck ./...
	gitleaks detect --source . --redact --no-banner
	shellcheck -e SC2016 install.sh scripts/*.sh tests/*.sh

ci: markdown budgets test vet lint actionlint security build smoke integration release-artifact-smoke release-check release-preflight

clean:
	rm -f $(BINARY) coverage.out

cover: test
	go tool cover -html=coverage.out

.DEFAULT_GOAL := build
