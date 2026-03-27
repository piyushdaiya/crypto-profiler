GO ?= go
TOOLS_BIN ?= $(CURDIR)/.tools/bin
GOCACHE ?= /tmp/crypto-profiler-go-cache
GOMODCACHE ?= /tmp/crypto-profiler-go-mod
GOVULNCHECK_VERSION ?= v1.1.4
GOSEC_VERSION ?= v2.22.4

.PHONY: build test test-verbose security security-tools govulncheck gosec

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) build ./...

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./...

test-verbose:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) test ./... -v

security-tools:
	mkdir -p $(TOOLS_BIN)
	GOBIN=$(TOOLS_BIN) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	GOBIN=$(TOOLS_BIN) GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)

govulncheck: security-tools
	PATH="$(TOOLS_BIN):$$PATH" govulncheck ./...

gosec: security-tools
	PATH="$(TOOLS_BIN):$$PATH" gosec ./...

security: govulncheck gosec
