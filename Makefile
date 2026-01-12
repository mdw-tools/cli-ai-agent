#!/usr/bin/make -f

VERSION := $(shell git describe)

test: fmt
	go test -race -cover -timeout=1s ./...

fmt:
	@go version && go fmt ./... && go mod tidy

install: test
	go install -ldflags="-X 'main.Version=$(VERSION)'" github.com/mdw-tools/cli-ai-agent/cmd/...
