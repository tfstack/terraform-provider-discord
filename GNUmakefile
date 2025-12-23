.PHONY: build install test test-coverage testacc fmt docs clean test-e2e

# Default target
.DEFAULT_GOAL := help

# Build the provider
build:
	@echo "==> Building the provider..."
	go build -buildvcs=false -o terraform-provider-discord

# Install the provider
install: build
	@echo "==> Installing the provider..."
	go install

# Install provider locally for Terraform to use
install-local: build
	@echo "==> Installing provider locally for Terraform..."
	@VERSION="0.1.0" \
	PLATFORM="linux_amd64" \
	PLUGIN_DIR="$$HOME/.terraform.d/plugins/registry.terraform.io/tfstack/discord/$$VERSION/$$PLATFORM" \
	&& mkdir -p "$$PLUGIN_DIR" \
	&& cp terraform-provider-discord "$$PLUGIN_DIR/" \
	&& echo "✅ Provider installed to $$PLUGIN_DIR"

# Run tests
test:
	@echo "==> Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "==> Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	@echo "==> Coverage report:"
	@go tool cover -func=coverage.out
	@echo ""
	@echo "==> HTML coverage report generated: coverage.html"
	@go tool cover -html=coverage.out -o coverage.html

# Run acceptance tests
testacc:
	@echo "==> Running acceptance tests..."
	TF_ACC=1 go test -v ./...

# Format code
fmt:
	@echo "==> Formatting code..."
	go fmt ./...
	terraform fmt -recursive ./examples/

# Generate documentation
docs:
	@echo "==> Generating documentation..."
	go generate ./...

# Initialize Terraform in all examples
init-examples: install-local
	@echo "==> Initializing Terraform in examples..."
	@for dir in examples/data-sources/*/ examples/resources/*/ examples/provider/; do \
		if [ -f "$$dir/data-source.tf" ] || [ -f "$$dir/resource.tf" ] || [ -f "$$dir/provider.tf" ] || [ -f "$$dir/main.tf" ] || [ -f "$$dir"*.tf ]; then \
			echo "Initializing $$dir..."; \
			cd "$$dir" && terraform init -upgrade > /dev/null 2>&1 && echo "✅ $$dir initialized" || echo "⚠️  $$dir skipped (may need variables)"; \
			cd - > /dev/null; \
		fi \
	done

# Initialize a specific example
init-example: install-local
	@if [ -z "$(EXAMPLE)" ]; then \
		echo "Usage: make init-example EXAMPLE=examples/data-sources/discord_channels"; \
		exit 1; \
	 fi
	@echo "==> Initializing $(EXAMPLE)..."
	@cd $(EXAMPLE) && terraform init -upgrade

# Clean build artifacts
clean:
	@echo "==> Cleaning..."
	rm -f terraform-provider-discord
	rm -f terraform-provider-discord.exe
	rm -f coverage.out coverage.html
	go clean
	@echo "==> Cleaning Terraform state files..."
	@find examples -name ".terraform" -type d -exec rm -rf {} + 2>/dev/null || true
	@find examples -name ".terraform.lock.hcl" -type f -delete 2>/dev/null || true
	@find examples -name "*.tfstate" -type f -delete 2>/dev/null || true
	@find examples -name "*.tfstate.*" -type f -delete 2>/dev/null || true

# Help target
help:
	@echo "Available targets:"
	@echo "  build          - Build the provider binary"
	@echo "  install        - Install the provider to GOPATH/bin"
	@echo "  install-local   - Install provider locally for Terraform testing"
	@echo "  init-examples   - Initialize Terraform in all examples (auto-installs provider)"
	@echo "  init-example    - Initialize a specific example (use EXAMPLE=path)"
	@echo "  test            - Run unit tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  testacc         - Run acceptance tests (requires TF_ACC=1)"
	@echo "  test-e2e        - Run end-to-end validation tests"
	@echo "  fmt             - Format code"
	@echo "  docs            - Generate documentation"
	@echo "  clean           - Clean build artifacts and Terraform state"
	@echo "  help            - Show this help message"
