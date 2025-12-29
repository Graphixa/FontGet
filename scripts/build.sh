#!/bin/bash
# FontGet Build Script for Linux/macOS
# Automatically detects version from git tags

set -e

VERSION=""
DEV=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        --dev)
            DEV=true
            shift
            ;;
        -h|--help)
            echo "FontGet Build Script"
            echo ""
            echo "Usage: ./scripts/build.sh [options]"
            echo ""
            echo "Options:"
            echo "  -v, --version <version>  Build with specific version (e.g., 2.0.0)"
            echo "  --dev                    Build as development version (latest release + commit hash)"
            echo "  -h, --help               Show this help message"
            echo ""
            echo "Examples:"
            echo "  ./scripts/build.sh              # Auto-detect version from git tag"
            echo "  ./scripts/build.sh --dev         # Build as 'dev' version"
            echo "  ./scripts/build.sh -v 2.1.0      # Build with specific version"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use -h or --help for usage information"
            exit 1
            ;;
    esac
done

# Get git info
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Determine version
if [ "$DEV" = true ]; then
    # For dev builds: get latest release version + commit hash
    TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$TAG" ]; then
        BASE_VERSION=$(echo "$TAG" | sed 's/^v//')
        VERSION="$BASE_VERSION-dev+$COMMIT"
        echo "Building FontGet (dev build: $VERSION)..."
    else
        # No tag found, use plain dev+commit
        VERSION="dev+$COMMIT"
        echo "Building FontGet (dev build: $VERSION - no release tag found)..."
    fi
elif [ -n "$VERSION" ]; then
    echo "Building FontGet v$VERSION (release)..."
else
    # Auto-detect from git tag
    TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    if [ -n "$TAG" ]; then
        VERSION=$(echo "$TAG" | sed 's/^v//')
        echo "Building FontGet v$VERSION (auto-detected from git tag)..."
    else
        VERSION="dev"
        echo "Building FontGet (dev build - no git tag found)..."
    fi
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

