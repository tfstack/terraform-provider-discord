#!/bin/bash
set -e

echo "🧪 Running End-to-End Tests for Terraform Provider Discord"
echo "============================================================"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test directory
TEST_DIR="test-e2e"
cd "$(dirname "$0")/.."

# Step 1: Build provider
echo ""
echo "📦 Step 1: Building provider..."
if make build > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Provider built successfully${NC}"
else
    echo -e "${RED}❌ Provider build failed${NC}"
    exit 1
fi

# Step 2: Install provider locally
echo ""
echo "📦 Step 2: Installing provider locally..."
if make install-local > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Provider installed successfully${NC}"
else
    echo -e "${RED}❌ Provider installation failed${NC}"
    exit 1
fi

# Step 3: Initialize test directory
echo ""
echo "📦 Step 3: Initializing test Terraform configuration..."
cd "$TEST_DIR"
if terraform init -upgrade > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Terraform initialized successfully${NC}"
else
    echo -e "${RED}❌ Terraform initialization failed${NC}"
    exit 1
fi

# Step 4: Validate configuration
echo ""
echo "🔍 Step 4: Validating Terraform configuration..."
if terraform validate > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Configuration is valid${NC}"
else
    echo -e "${YELLOW}⚠️  Configuration validation failed (may need variables)${NC}"
    terraform validate
fi

# Step 5: Check if we have credentials
echo ""
echo "🔐 Step 5: Checking credentials..."
if [ -z "$DISCORD_BOT_TOKEN" ]; then
    echo -e "${YELLOW}⚠️  DISCORD_BOT_TOKEN not set - skipping live tests${NC}"
    echo -e "${YELLOW}   Set DISCORD_BOT_TOKEN and TF_VAR_guild_id to run live tests${NC}"
    echo ""
    echo -e "${GREEN}✅ Static validation tests passed!${NC}"
    exit 0
fi

if [ -z "$TF_VAR_guild_id" ]; then
    echo -e "${YELLOW}⚠️  TF_VAR_guild_id not set - skipping live tests${NC}"
    echo -e "${YELLOW}   Set TF_VAR_guild_id to run live tests${NC}"
    echo ""
    echo -e "${GREEN}✅ Static validation tests passed!${NC}"
    exit 0
fi

# Step 6: Run terraform plan (dry run)
echo ""
echo "📋 Step 6: Running terraform plan (dry run)..."
if terraform plan -out=tfplan > /tmp/terraform-plan.log 2>&1; then
    echo -e "${GREEN}✅ Plan succeeded${NC}"
    echo ""
    echo "Plan summary:"
    grep -E "(will be created|will be destroyed|will be updated)" /tmp/terraform-plan.log | head -10 || echo "No changes planned"
else
    echo -e "${RED}❌ Plan failed${NC}"
    cat /tmp/terraform-plan.log
    exit 1
fi

echo ""
echo -e "${GREEN}✅ All end-to-end tests passed!${NC}"
echo ""
echo "To apply changes, run:"
echo "  cd $TEST_DIR"
echo "  terraform apply"
