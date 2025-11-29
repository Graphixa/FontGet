#!/bin/sh
#
# FontGet Installer Script
# Installs FontGet CLI tool on Linux and macOS
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
#   # Or with a specific version:
#   FONTGET_VERSION=1.0.0 curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
#

set -e

# Colors for output (if terminal supports it)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Repository information
REPO="Graphixa/FontGet"
REPO_URL="https://github.com/${REPO}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo "${RED}Error: Unsupported operating system: $OS${NC}" >&2
    echo "FontGet supports Linux and macOS only." >&2
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "${RED}Error: Unsupported architecture: $ARCH${NC}" >&2
        exit 1
        ;;
esac

# Determine version to install
VERSION="${FONTGET_VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
    BASE_URL="${REPO_URL}/releases/latest/download"
    echo "${BLUE}Installing latest version of FontGet...${NC}"
else
    # Remove 'v' prefix if present
    VERSION=$(echo "$VERSION" | sed 's/^v//')
    BASE_URL="${REPO_URL}/releases/download/v${VERSION}"
    echo "${BLUE}Installing FontGet v${VERSION}...${NC}"
fi

# Binary name (no extension for Linux/macOS)
BINARY_NAME="fontget-${OS}-${ARCH}"
DOWNLOAD_URL="${BASE_URL}/${BINARY_NAME}"

# Installation directory
INSTALL_DIR="${FONTGET_INSTALL_DIR:-${HOME}/.local/bin}"
mkdir -p "$INSTALL_DIR"

# Check if fontget is already installed
INSTALLED_BIN="${INSTALL_DIR}/fontget"
if [ -f "$INSTALLED_BIN" ]; then
    CURRENT_VERSION=$("$INSTALLED_BIN" version 2>/dev/null | head -n1 | sed 's/.*v\([0-9.]*\).*/\1/' || echo "unknown")
    echo "${YELLOW}FontGet is already installed at: $INSTALLED_BIN${NC}"
    echo "${YELLOW}Current version: v${CURRENT_VERSION}${NC}"
    echo "${YELLOW}This will be overwritten.${NC}"
    echo ""
fi

# Create temporary directory for download
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download binary
echo "${BLUE}Downloading FontGet...${NC}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "${TMP_DIR}/fontget"; then
    echo "${RED}Error: Failed to download FontGet${NC}" >&2
    if [ "$VERSION" != "latest" ]; then
        echo "${YELLOW}Version v${VERSION} may not exist. Check available versions at:${NC}"
        echo "${BLUE}${REPO_URL}/releases${NC}"
    fi
    exit 1
fi

# Make binary executable
chmod +x "${TMP_DIR}/fontget"

# Verify binary works (basic check)
if ! "${TMP_DIR}/fontget" version >/dev/null 2>&1; then
    echo "${RED}Error: Downloaded binary appears to be invalid${NC}" >&2
    exit 1
fi

# Install binary
echo "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
mv "${TMP_DIR}/fontget" "$INSTALLED_BIN"

# Get installed version
INSTALLED_VERSION=$("$INSTALLED_BIN" version 2>/dev/null | head -n1 || echo "FontGet")
echo ""
echo "${GREEN}✓ FontGet installed successfully!${NC}"
echo ""
echo "  Location: ${INSTALLED_BIN}"
echo "  Version:  ${INSTALLED_VERSION}"
echo ""

# Check if install directory is in PATH
case ":$PATH:" in
    *:"${INSTALL_DIR}":*)
        echo "${GREEN}✓ ${INSTALL_DIR} is already in your PATH${NC}"
        ;;
    *)
        echo "${YELLOW}⚠ ${INSTALL_DIR} is not in your PATH${NC}"
        echo ""
        echo "To use FontGet, add this to your shell profile:"
        if [ -n "$ZSH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"\${HOME}/.local/bin:\${PATH}\"' >> ~/.zshrc${NC}"
            echo "  ${BLUE}source ~/.zshrc${NC}"
        elif [ -n "$BASH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"\${HOME}/.local/bin:\${PATH}\"' >> ~/.bashrc${NC}"
            echo "  ${BLUE}source ~/.bashrc${NC}"
        else
            echo "  ${BLUE}export PATH=\"\${HOME}/.local/bin:\${PATH}\"${NC}"
        fi
        echo ""
        echo "Or run FontGet directly:"
        echo "  ${BLUE}${INSTALLED_BIN} --help${NC}"
        ;;
esac

echo ""
echo "${GREEN}You can now use 'fontget' to manage your fonts!${NC}"
echo "  ${BLUE}fontget search \"roboto\"${NC}"
echo "  ${BLUE}fontget add google.roboto${NC}"
echo ""

