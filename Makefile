BINARY  ?= garm-provider-gcp-mig
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0-dev)
LDFLAGS := -s -w -X github.com/ut0mt8/garm-provider-gcp-mig/provider.Version=$(VERSION)

.PHONY: build test lint tidy release clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

test:
	go test -race ./...

lint:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))" || (gofmt -l $$(git ls-files '*.go'); exit 1)
	go vet ./...

tidy:
	go mod tidy

# Cross-compiled release archives in dist/
release:
	rm -rf dist && mkdir -p dist
	for arch in amd64 arm64; do \
	  CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$(BINARY) . && \
	  tar -C dist -czf dist/$(BINARY)_$(VERSION)_linux_$$arch.tar.gz $(BINARY) && rm dist/$(BINARY); \
	done
	cd dist && sha256sum *.tar.gz > SHA256SUMS

clean:
	rm -rf bin dist
