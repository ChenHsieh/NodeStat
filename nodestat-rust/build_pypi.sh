#!/bin/bash
# Build script for Rust + PyPI package

set -e

echo "🦀 Building NodeStat Rust + PyPI package..."

# Check requirements
if ! command -v cargo &> /dev/null; then
    echo "❌ Cargo/Rust is required"
    exit 1
fi

if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is required"
    exit 1
fi

# Build Rust binary
echo "🔨 Building Rust binary..."
cargo build --release

# Copy binary to Python package
echo "📦 Packaging for PyPI..."
mkdir -p python/nodestat_tui/
cp target/release/nodestat python/nodestat_tui/nodestat
chmod +x python/nodestat_tui/nodestat

echo "✅ Build complete!"
echo ""
echo "🚀 To test locally:"
echo "  cd python && pip install -e ."
echo "  nodestat -s mock"
echo ""
echo "📦 To build wheel for PyPI:"
echo "  pip install build"
echo "  python -m build"
echo ""
echo "🌍 To upload to PyPI:"
echo "  pip install twine"
echo "  twine upload dist/*"