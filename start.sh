#!/bin/bash

# z9s Build Script
# Executes go mod tidy and builds the z9s binary

set -e  # Exit on error

echo "🔨 Building z9s..."
echo ""

# Run go mod tidy
echo "📦 Running go mod tidy..."
go mod tidy

if [ $? -ne 0 ]; then
    echo "❌ go mod tidy failed"
    exit 1
fi

echo "✅ Dependencies resolved"
echo ""

# Build the binary
echo "🏗️  Building z9s binary..."
go build -o z9s

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful!"
echo ""
echo "🚀 z9s binary ready at: ./z9s"
echo ""
echo "To run: ./z9s"
