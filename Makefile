.PHONY: build install clean test fmt lint

BINARY_NAME=wt
BUILD_DIR=./build
INSTALL_DIR=$(HOME)/.local/bin

# Build for current platform
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/wt

# Install to ~/.local/bin
install: build
	mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	codesign -s - $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to $(INSTALL_DIR)"

# Uninstall
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from $(INSTALL_DIR)"

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	go clean

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Initialize config (convenience target)
init: install
	$(INSTALL_DIR)/$(BINARY_NAME) init

# Development: build and run
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Development: build and show help
help: build
	$(BUILD_DIR)/$(BINARY_NAME) --help
