BINARY := crmservice
BUILD_DIR := build
PREFIX := $(if $(filter 0,$(shell id -u)),/usr/local,$(HOME)/.local)
BINDIR := $(PREFIX)/bin

.PHONY: build install test clean

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .

install: build
	mkdir -p $(BINDIR)
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(BINDIR)/$(BINARY)

test:
	go test ./...

clean:
	rm -rf $(BUILD_DIR)
