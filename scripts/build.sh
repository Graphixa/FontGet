#!/bin/bash
# FontGet Build Script for Linux/macOS
# Simple build script for local testing
# Note: Release builds are handled automatically by GitHub Actions on tag push

set -e

# Run from repo root (so script works when called from any directory)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

VERSION=""

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            echo "FontGet Build Script"
            echo ""
            echo "Usage: ./scripts/build.sh [options]"
            echo ""
            echo "Options:"
            echo "  -v, --version <version>  Build with specific version (for testing release builds locally)"
            echo "  -h, --help               Show this help message"
            echo ""
            echo "Examples:"
            echo "  ./scripts/build.sh              # Build for local testing (uses 'dev' version)"
            echo "  ./scripts/build.sh -v 2.1.0      # Test a specific version locally"
            echo ""
            echo "Note: For releases, just create and push a git tag. GitHub Actions will build automatically."
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Get git info (for build metadata)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Determine version
if [ -n "$VERSION" ]; then
    # User specified a version (for testing release builds locally)
    echo "Building FontGet v$VERSION (local test build)..."
else
    # Default: simple dev build for local testing
    VERSION="dev"
    echo "Building FontGet (local dev build)..."
fi

echo "  Version: $VERSION"
echo "  Commit:  $COMMIT"
echo "  Date:    $DATE"
echo ""

# Build flags
LDFLAGS="-s -w \
    -X fontget/internal/version.Version=$VERSION \
    -X fontget/internal/version.GitCommit=$COMMIT \
    -X fontget/internal/version.BuildDate=$DATE"

# Build
go build -ldflags "$LDFLAGS" -o fontget .

if [ $? -eq 0 ]; then
    echo ""
    echo "Build successful! Binary: ./fontget"
    echo ""
    echo "Version info:"
    ./fontget version
else
    echo ""
    echo "Build failed!"
    exit 1
fi

