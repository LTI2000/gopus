# Makefile for gopus

# Binary name
BINARY_NAME := gopus

#GOFLAGS := -v

# Generated files
GEN_CLIENT := internal/openai/client_gen.go
GEN_MODELS := internal/openai/models_gen.go

.PHONY: all generate build clean run

all: generate build

generate:
	go generate $(GOFLAGS) ./...

build:
	go build $(GOFLAGS) -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)
	rm -f $(GEN_CLIENT)
	rm -f $(GEN_MODELS)

run: clean all
	./$(BINARY_NAME)
