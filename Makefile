.PHONY: all build install test clean

BINARY_NAME := vast
INSTALL_DIR := $(HOME)/.local/bin
LDFLAGS := -ldflags="-s -w"

all: test build

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/vast
	ln -sf $(BINARY_NAME) bin/vastai

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 755 bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	ln -sf $(BINARY_NAME) $(INSTALL_DIR)/vastai
	@echo "Installed $(BINARY_NAME) and vastai to $(INSTALL_DIR)"

test:
	go test -v ./...

clean:
	rm -rf bin
