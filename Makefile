# Go 1.22 install; GOROOT must match or a shell-level GOROOT (e.g. old Go) breaks the build.
export GOROOT := $(HOME)/sdk/go1.22.12
GO ?= $(GOROOT)/bin/go

.PHONY: build test lint run

build:
	$(GO) build -o bin/todo ./cmd/todo

test:
	$(GO) test -race ./...

# golangci-lint is used when installed; otherwise gofmt + go vet only.
lint:
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	$(GO) vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run ./...; else echo "golangci-lint not installed, skipped"; fi

run:
	$(GO) run ./cmd/todo
