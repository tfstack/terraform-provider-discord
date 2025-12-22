#!/bin/bash
set -e

echo "🚀 Setting up Terraform Provider Discord development environment..."

# Display system information
echo "📋 System Information:"
uname -a
echo "Go version: $(go version)"
echo "Go path: $(go env GOPATH)"

# Install Terraform
echo "📦 Installing Terraform..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y apt-utils gnupg software-properties-common

curl -fsSL https://apt.releases.hashicorp.com/gpg | gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/hashicorp.list
apt-get update -y
apt-get install -y terraform

echo "✅ Terraform installed: $(terraform version)"

# Install Terraform Plugin Framework docs generator
echo "📚 Installing Terraform Plugin Framework documentation generator..."
go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

# Verify Go tools
echo "🔧 Verifying Go tools..."
echo "golangci-lint: $(golangci-lint version)"
echo "goimports: $(which goimports)"
echo "gopls: $(which gopls)"

# Download Go dependencies
echo "📥 Downloading Go dependencies..."
cd /workspaces/terraform-provider-discord || cd /workspace
go mod download
go mod verify

# Build the provider to verify everything works
echo "🔨 Building provider..."
go build -buildvcs=false -o terraform-provider-discord

# Install provider locally for Terraform to use
echo "📦 Installing provider locally for Terraform..."
VERSION="0.1.0"
PLATFORM="linux_amd64"
PLUGIN_DIR="${HOME}/.terraform.d/plugins/registry.terraform.io/tfstack/discord/${VERSION}/${PLATFORM}"
mkdir -p "${PLUGIN_DIR}"
cp terraform-provider-discord "${PLUGIN_DIR}/"
echo "✅ Provider installed to ${PLUGIN_DIR}"

# Initialize Terraform in examples (non-blocking, may fail if variables needed)
echo "🔧 Initializing Terraform examples..."
for dir in examples/data-sources/*/ examples/resources/*/ examples/provider/; do
	if [ -f "${dir}data-source.tf" ] || [ -f "${dir}resource.tf" ] || [ -f "${dir}provider.tf" ] || [ -f "${dir}main.tf" ] || [ -f "${dir}"*.tf ]; then
		echo "  Initializing ${dir}..."
		cd "${dir}" && terraform init -upgrade > /dev/null 2>&1 && echo "    ✅ ${dir} initialized" || echo "    ⚠️  ${dir} skipped (may need variables)"
		cd - > /dev/null
	fi
	done

# Load .env file if it exists
echo "🔐 Loading environment variables from .env file..."
if [ -f /workspaces/terraform-provider-discord/.env ]; then
    set -a
    source /workspaces/terraform-provider-discord/.env
    set +a
    echo "✅ Environment variables loaded from .env"
elif [ -f /workspace/.env ]; then
    set -a
    source /workspace/.env
    set +a
    echo "✅ Environment variables loaded from .env"
else
    echo "⚠️  No .env file found. Create one from .env.example if needed."
fi

echo ""
echo "✅ Development environment setup complete!"
echo ""
echo "Available commands:"
echo "  make build          - Build the provider"
echo "  make install        - Install the provider"
echo "  make install-local   - Install provider locally for Terraform testing"
echo "  make init-examples  - Initialize Terraform in all examples"
echo "  make init-example   - Initialize a specific example (EXAMPLE=path)"
echo "  make test           - Run tests"
echo "  make fmt            - Format code"
echo "  make docs           - Generate documentation"
echo ""
echo "💡 The provider is already installed locally and examples are initialized!"
echo "   Navigate to any example directory and run 'terraform plan' or 'terraform apply'."
