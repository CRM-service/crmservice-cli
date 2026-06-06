BINARY := crmservice
BUILD_DIR := build
PREFIX := $(if $(filter 0,$(shell id -u)),/usr/local,$(HOME)/.local)
BINDIR := $(PREFIX)/bin

GOLANGCI_LINT_VERSION := v1.64.8

.PHONY: build install test lint check clean

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .

install: build
	mkdir -p $(BINDIR)
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(BINDIR)/$(BINARY)

test:
	go test ./...

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

check: lint test

clean:
	rm -rf $(BUILD_DIR)
