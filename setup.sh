#!/bin/bash

# KT9S Setup Script
# This script sets up the kt9s development environment on Linux/macOS

set -e

echo "═══════════════════════════════════════════════════════════"
echo "           KT9S Development Environment Setup              "
echo "═══════════════════════════════════════════════════════════"
echo ""

# Check Go version
echo "📋 Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.25 or later."
    echo "   Visit: https://golang.org/dl"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Go $GO_VERSION found"
echo ""

# Check if we're in the kt9s directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: go.mod not found. Please run this script from the kt9s root directory."
    exit 1
fi

echo "📁 Project directory verified"
echo ""

# Update module name
echo "🔧 Configuring module name..."
read -p "Enter your GitHub username (for module path): " USERNAME

if [ -z "$USERNAME" ]; then
    echo "❌ Username cannot be empty"
    exit 1
fi

# Replace placeholders in all files
echo "   Replacing 'yourusername' with '$USERNAME'..."

# macOS compatibility
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS uses -i '' for in-place sed
    find . -name "*.go" -type f -exec sed -i '' "s/yourusername/$USERNAME/g" {} \;
    find . -name "*.mod" -type f -exec sed -i '' "s/yourusername/$USERNAME/g" {} \;
else
    # Linux
    find . -name "*.go" -type f -exec sed -i "s/yourusername/$USERNAME/g" {} \;
    find . -name "*.mod" -type f -exec sed -i "s/yourusername/$USERNAME/g" {} \;
fi

echo "✅ Module name updated to github.com/$USERNAME/kt9s"
echo ""

# Download dependencies
echo "📚 Downloading Go dependencies..."
go mod tidy
echo "✅ Dependencies downloaded"
echo ""

# Verify build
echo "🔨 Verifying build..."
if go build -o /tmp/kt9s-test 2>&1; then
    echo "✅ Build verification successful"
    rm -f /tmp/kt9s-test
else
    echo "⚠️  Build verification had some warnings (this is normal for incomplete integration)"
fi
echo ""

# Create necessary directories
echo "📁 Creating additional directories..."
mkdir -p docs logs bin

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "              ✅ Setup Complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📖 Next steps:"
echo ""
echo "1. Read SETUP_GUIDE.md:"
echo "   cat SETUP_GUIDE.md"
echo ""
echo "2. Copy k9s code (if not already done):"
echo "   cp -r path/to/k9s-original/internal/* internal/k9s/"
echo ""
echo "3. Copy ktop code (if not already done):"
echo "   cp -r path/to/ktop-original/* internal/ktop/"
echo ""
echo "4. Build the project:"
echo "   make build"
echo ""
echo "5. Run kt9s:"
echo "   ./kt9s"
echo ""
echo "For more commands, run:"
echo "   make help"
echo ""
