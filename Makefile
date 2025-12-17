.PHONY: help install test lint fmt clean build-examples
.PHONY: run-openai-getting-started run-openai-basic run-openai-streaming run-openai-metadata
.PHONY: run-azure-getting-started run-azure-basic run-azure-streaming run-azure-metadata

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-30s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	go mod download
	go mod tidy

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	go vet ./...
	go fmt ./...

fmt: ## Format code
	go fmt ./...

clean: ## Clean build artifacts
	go clean
	rm -rf bin/

# OpenAI Examples
run-openai-getting-started: ## Run OpenAI getting started example
	go run examples/openai/getting-started/main.go

run-openai-basic: ## Run OpenAI basic example
	go run examples/openai/basic/main.go

run-openai-streaming: ## Run OpenAI streaming example
	go run examples/openai/streaming/main.go

run-openai-metadata: ## Run OpenAI metadata example
	go run examples/openai/metadata/main.go

# Azure OpenAI Examples
run-azure-getting-started: ## Run Azure OpenAI getting started example
	go run examples/azure/getting-started/main.go

run-azure-basic: ## Run Azure OpenAI basic example
	go run examples/azure/basic/main.go

run-azure-streaming: ## Run Azure OpenAI streaming example
	go run examples/azure/streaming/main.go

run-azure-metadata: ## Run Azure OpenAI metadata example
	go run examples/azure/metadata/main.go

build-examples: ## Build all examples
	@mkdir -p bin/openai bin/azure
	go build -o bin/openai/getting-started examples/openai/getting-started/main.go
	go build -o bin/openai/basic examples/openai/basic/main.go
	go build -o bin/openai/streaming examples/openai/streaming/main.go
	go build -o bin/openai/metadata examples/openai/metadata/main.go
	go build -o bin/azure/getting-started examples/azure/getting-started/main.go
	go build -o bin/azure/basic examples/azure/basic/main.go
	go build -o bin/azure/streaming examples/azure/streaming/main.go
	go build -o bin/azure/metadata examples/azure/metadata/main.go

