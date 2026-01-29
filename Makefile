# Makefile for gopus

# Binary name
BINARY_NAME := gopus

# Go commands
GO := go
#GOFLAGS := -v

# Generated files
GEN_CLIENT := internal/openai/client_gen.go
GEN_MODELS := internal/openai/models_gen.go

.PHONY: all build clean generate help

# Default target
all: generate build

# Build the binary
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

# Clean build artifacts and generated files
clean:
	rm -f $(BINARY_NAME)
	rm -f $(GEN_CLIENT)
	rm -f $(GEN_MODELS)

# Generate all code
generate: generate-models generate-client
	$(GO) generate $(GOFLAGS) ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Generate code and build (default)"
	@echo "  build            - Build the binary"
	@echo "  clean            - Remove build artifacts and generated files"
	@echo "  generate         - Generate all code from OpenAPI spec"
	@echo "  help             - Show this help message"
