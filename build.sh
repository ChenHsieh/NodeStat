#!/bin/bash
# Build script for NodeStat TUI

set -e

echo "🏗️  Building NodeStat TUI..."

# Check Go version
if ! command -v go &> /dev/null; then
    echo "❌ Go is required but not installed"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.20"

if ! printf '%s\n%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V -C; then
    echo "❌ Go $REQUIRED_VERSION or later is required (found $GO_VERSION)"
    exit 1
fi

echo "✅ Go $GO_VERSION found"

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -f nodestat nodestat.exe

# Download dependencies
echo "📦 Downloading dependencies..."
go mod tidy

# Build for current platform
echo "🔨 Building for $(go env GOOS)/$(go env GOARCH)..."
go build -ldflags="-s -w" -o nodestat .

# Make executable
chmod +x nodestat

echo "✅ Build complete!"
echo ""
echo "🚀 Quick start:"
echo "  ./nodestat -h          # Show help"
echo "  ./nodestat -s mock     # Demo mode"
echo "  ./nodestat -q batch    # Monitor batch partition"
echo ""
echo "🎯 Binary: $(pwd)/nodestat"
echo "📏 Size: $(du -h nodestat | awk '{print $1}')"

# Optional: Build for multiple platforms
if [[ "${1:-}" == "all" ]]; then
    echo ""
    echo "🌍 Building for multiple platforms..."
    
    platforms=(
        "linux/amd64"
        "linux/arm64"  
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    mkdir -p dist
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r os arch <<< "$platform"
        output="dist/nodestat-${os}-${arch}"
        
        if [[ "$os" == "windows" ]]; then
            output="${output}.exe"
        fi
        
        echo "  📦 Building $platform -> $output"
        GOOS="$os" GOARCH="$arch" go build -ldflags="-s -w" -o "$output" .
    done
    
    echo "✅ Multi-platform builds complete in dist/"
    ls -la dist/
fi