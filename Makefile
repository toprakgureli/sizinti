GO ?= go
EXE :=
ifeq ($(OS),Windows_NT)
EXE := .exe
endif

.PHONY: build test lint run bench fmt

build:
	$(GO) build -o bin/sizinti$(EXE) ./cmd/sizinti

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	staticcheck ./...
	golangci-lint run

run:
	$(GO) run ./cmd/sizinti $(ARGS)

bench:
	$(GO) test ./internal/scanner -run '^$$' -bench . -benchmem

fmt:
	gofmt -w .
	goimports -local github.com/toprakgureli/sizinti -w .
