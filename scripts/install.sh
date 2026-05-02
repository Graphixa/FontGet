#!/bin/sh
#
# FontGet Installer Script
# Installs FontGet CLI tool on Linux and macOS
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --dry-run
#   FONTGET_VERSION=1.0.0 curl -fsSL .../install.sh | sh
#
# Environment:
#   FONTGET_VERSION          Version or latest (default: latest)
#   FONTGET_INSTALL_DIR      Install directory (default: ~/.local/bin)
#   FONTGET_NONINTERACTIVE=1 Skip "Continue?" (non-interactive install)
#   FONTGET_DRY_RUN=1        Print plan only; no download or install (same as --dry-run)
#   CI                       When non-empty, prompt is skipped (common on CI runners)
#   NO_COLOR=1               Disable ANSI colors
#
#

set -e

DRY_RUN=false
for _arg in "$@"; do
    case "$_arg" in
        --dry-run|--dryrun)
            DRY_RUN=true
            ;;
        --help|-h)
            cat <<'EOF'
FontGet install.sh

  curl -fsSL https://raw.githubusercontent.com/Graphixa/FontGet/main/scripts/install.sh | sh
  curl -fsSL .../install.sh | sh -s -- --dry-run

Flags:
  --dry-run       Print URLs and paths only; no download or install
  --help, -h      This message

Environment:
  FONTGET_VERSION, FONTGET_INSTALL_DIR, FONTGET_NONINTERACTIVE=1, FONTGET_DRY_RUN=1
  NO_COLOR=1
EOF
            exit 0
            ;;
    esac
done
case "${FONTGET_DRY_RUN:-}" in 1|true|TRUE|yes|YES) DRY_RUN=true ;; esac

# Colors (respect NO_COLOR). printf gives real ESC; '\033' in quotes is literal.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
    ESC=$(printf '\033')
    RED="${ESC}[0;31m"
    GREEN="${ESC}[0;32m"
    YELLOW="${ESC}[1;33m"
    BLUE="${ESC}[94m"
    DIM="${ESC}[2m"
    NC="${ESC}[0m"
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    DIM=''
    NC=''
fi

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

VERSION="${FONTGET_VERSION:-latest}"
DISPLAY_VERSION="$VERSION"
if [ "$VERSION" != "latest" ]; then
    DISPLAY_VERSION=$(echo "$VERSION" | sed 's/^v//')
fi

if [ "$VERSION" = "latest" ]; then
    BASE_URL="${REPO_URL}/releases/latest/download"
else
    VERSION=$(echo "$VERSION" | sed 's/^v//')
    BASE_URL="${REPO_URL}/releases/download/v${VERSION}"
fi

BINARY_NAME="fontget-${OS}-${ARCH}"
DOWNLOAD_URL="${BASE_URL}/${BINARY_NAME}"
CHECKSUMS_URL="${BASE_URL}/checksums.txt"
INSTALL_DIR="${FONTGET_INSTALL_DIR:-${HOME}/.local/bin}"
INSTALLED_BIN="${INSTALL_DIR}/fontget"

# --- splash (default terminal foreground — no accent color on banner) ---
cat <<'SPLASH'

███████╗░█████╗░███╗░░██╗████████╗░██████╗░███████╗████████╗
██╔════╝██╔══██╗████╗░██║╚══██╔══╝██╔════╝░██╔════╝╚══██╔══╝
█████╗░░██║░░██║██╔██╗██║░░░██║░░░██║░░██╗░█████╗░░░░░██║░░░
██╔══╝░░██║░░██║██║╚████║░░░██║░░░██║░░╚██╗██╔══╝░░░░░██║░░░
██║░░░░░╚█████╔╝██║░╚███║░░░██║░░░╚██████╔╝███████╗░░░██║░░░
╚═╝░░░░░░╚════╝░╚═╝░░╚══╝░░░╚═╝░░░░╚═════╝░╚══════╝░░░╚═╝░░░

SPLASH

TAGLINE="Discover, install & manage fonts from the command line."
SPLASH_W=60
_taglen=$(printf '%s' "$TAGLINE" | wc -c | tr -d ' ')
if [ "$_taglen" -gt "$SPLASH_W" ]; then
    echo "${DIM}${TAGLINE}${NC}"
else
    _l=$(( (SPLASH_W - _taglen) / 2 ))
    _r=$(( SPLASH_W - _taglen - _l ))
    printf '%s%*s%s%*s%s\n' "${DIM}" "$_l" "" "$TAGLINE" "$_r" "" "${NC}"
fi
echo ""

echo "${BLUE}OS:${NC} ${OS} ${BLUE}|${NC} ${BLUE}ARCH:${NC} ${ARCH}"
echo ""
echo "This will install FontGet ${DISPLAY_VERSION} to ${INSTALLED_BIN}"
echo ""

# Existing install notice before prompt
if [ -f "$INSTALLED_BIN" ]; then
    CURRENT_VERSION=$("$INSTALLED_BIN" version 2>/dev/null | head -n1 | sed 's/.*v\([0-9.]*\).*/\1/' || echo "unknown")
    echo "${YELLOW}FontGet is already installed at: $INSTALLED_BIN${NC}"
    echo "${YELLOW}Current version: v${CURRENT_VERSION}${NC}"
    echo "${YELLOW}This will be overwritten.${NC}"
    echo ""
fi

if [ "$DRY_RUN" = true ]; then
    echo "${BLUE}[dry-run] No download or install will be performed.${NC}"
    echo "${BLUE}[dry-run] Download:${NC} ${DOWNLOAD_URL}"
    echo "${BLUE}[dry-run] Checksums:${NC} ${CHECKSUMS_URL}"
    echo "${BLUE}[dry-run] Install to:${NC} ${INSTALLED_BIN}"
    exit 0
fi

# Continue prompt: only when interactive; CI / non-TTY / NONINTERACTIVE skip (safe for curl | sh and automation)
SHOULD_PROMPT=true
if [ "${FONTGET_NONINTERACTIVE:-0}" = "1" ]; then
    SHOULD_PROMPT=false
elif [ ! -t 0 ] || [ ! -t 1 ]; then
    SHOULD_PROMPT=false
elif [ -n "${CI:-}" ]; then
    SHOULD_PROMPT=false
fi

if [ "$SHOULD_PROMPT" = true ]; then
    printf "Continue? [y/N] "
    read -r REPLY || REPLY=n
    case "$REPLY" in
        [yY]|[yY][eE][sS]) ;;
        *)
            echo "${YELLOW}Cancelled.${NC}"
            exit 0
            ;;
    esac
    echo ""
fi

mkdir -p "$INSTALL_DIR"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

LOCAL_BIN="${TMP_DIR}/${BINARY_NAME}"

echo "${BLUE}Downloading FontGet...${NC}"
if ! curl -fsSL "$DOWNLOAD_URL" -o "$LOCAL_BIN"; then
    echo "${RED}Error: Failed to download FontGet${NC}" >&2
    if [ "${DISPLAY_VERSION}" != "latest" ]; then
        echo "${YELLOW}Version v${DISPLAY_VERSION} may not exist. Check available versions at:${NC}"
        echo "${BLUE}${REPO_URL}/releases${NC}"
    fi
    exit 1
fi

echo "${BLUE}Downloading checksums...${NC}"
if ! curl -fsSL "$CHECKSUMS_URL" -o "${TMP_DIR}/checksums.txt"; then
    echo "${RED}Error: Failed to download checksums.txt${NC}" >&2
    exit 1
fi

EXPECTED=""
EXPECTED=$(tr -d '\r' < "${TMP_DIR}/checksums.txt" | grep -E "[[:space:]]${BINARY_NAME}\$" 2>/dev/null | head -n1 | awk '{print $1}' || true)
if [ -z "$EXPECTED" ]; then
    echo "${RED}Error: No checksum line for ${BINARY_NAME} in checksums.txt${NC}" >&2
    exit 1
fi

ACTUAL=""
if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "$LOCAL_BIN" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "$LOCAL_BIN" | awk '{print $1}')
else
    ACTUAL=$(openssl dgst -sha256 "$LOCAL_BIN" | awk '{print $NF}')
fi

EXL=$(printf '%s' "$EXPECTED" | tr '[:upper:]' '[:lower:]')
ACL=$(printf '%s' "$ACTUAL" | tr '[:upper:]' '[:lower:]')
if [ "$EXL" != "$ACL" ]; then
    echo "${RED}Error: Checksum mismatch for ${BINARY_NAME}${NC}" >&2
    exit 1
fi

echo "${GREEN}✓ Checksum verified${NC}"

chmod +x "$LOCAL_BIN"

if ! "$LOCAL_BIN" version >/dev/null 2>&1; then
    echo "${RED}Error: Downloaded binary appears to be invalid${NC}" >&2
    exit 1
fi

echo "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
mv "$LOCAL_BIN" "$INSTALLED_BIN"

INSTALLED_VERSION=$("$INSTALLED_BIN" version 2>/dev/null | head -n1 || echo "FontGet")
echo ""
echo "${GREEN}✓ FontGet installed successfully!${NC}"
echo ""
echo "  Location: ${INSTALLED_BIN}"
echo "  Version:  ${INSTALLED_VERSION}"
echo ""

case ":$PATH:" in
    *:"${INSTALL_DIR}":*)
        echo "${GREEN}✓ ${INSTALL_DIR} is already in your PATH${NC}"
        ;;
    *)
        echo "${YELLOW}⚠ ${INSTALL_DIR} is not in your PATH${NC}"
        echo ""
        echo "To use FontGet, add this to your shell profile:"
        # Detect shell and recommend proper config file addition
        if [ -n "$ZSH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"${INSTALL_DIR}:\${PATH}\"' >> ~/.zshrc${NC}"
            echo "  ${BLUE}source ~/.zshrc${NC}"
        elif [ -n "$BASH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"${INSTALL_DIR}:\${PATH}\"' >> ~/.bashrc${NC}"
            echo "  ${BLUE}source ~/.bashrc${NC}"
        elif [ -n "$FISH_VERSION" ]; then
            echo "  ${BLUE}set -U fish_user_paths ${INSTALL_DIR} \$fish_user_paths${NC}"
            echo "  ${BLUE}exec fish${NC}"
        elif [ -n "$KSH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"${INSTALL_DIR}:\${PATH}\"' >> ~/.kshrc${NC}"
            echo "  ${BLUE}source ~/.kshrc${NC}"
        elif [ -n "$POSH_VERSION" ]; then
            echo "  ${BLUE}echo 'export PATH=\"${INSTALL_DIR}:\${PATH}\"' >> ~/.profile${NC}"
            echo "  ${BLUE}source ~/.profile${NC}"
        else
            echo "  ${BLUE}export PATH=\"${INSTALL_DIR}:\${PATH}\"${NC}"
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
