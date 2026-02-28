#!/bin/sh
#
# FontGet build script (Linux/macOS).
# Run with:  sh scripts/build.sh   (no execute bit needed)
#
# By default builds to /tmp/fontget-dev (a file) so the binary always runs (e.g. on cloud drives).
# Override with FONTGET_OUTPUT=./fontget to build in the repo.
#

set -e

# Resolve repo root (script lives in scripts/, repo is parent)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

# Default: build to /tmp/fontget
OUTPUT="${FONTGET_OUTPUT:-/tmp/fontget}"

# Parse -v / --version and -h / --help
VERSION=""
while [ $# -gt 0 ]; do
    case "$1" in
        -v|--version)
            VERSION="${2:-}"
            shift 2
            ;;
        -h|--help)
            echo "FontGet build script"
            echo ""
            echo "  sh scripts/build.sh           # build (dev version) → /tmp/fontget-dev"
            echo "  sh scripts/build.sh -v 2.1.0   # build with version string"
            echo "  FONTGET_OUTPUT=./fontget sh scripts/build.sh   # build in repo"
            echo ""
            echo "Default output: /tmp/fontget (so the binary runs on all drives)."
            echo "Override: FONTGET_OUTPUT=/path/to/fontget"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            echo "Use -h for help." >&2
            exit 1
            ;;
    esac
done

COMMIT=$(git rev-parse --short=12 HEAD 2>/dev/null) || COMMIT=$(git rev-parse --short HEAD 2>/dev/null) || COMMIT="unknown"
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
DATE_COMPACT=$(date -u +"%Y%m%d%H%M%S")

# Dev builds: dev-YYYYMMDDHHMMSS-<commit> so they sort lower than release versions
[ -z "$VERSION" ] && VERSION="dev-${DATE_COMPACT}-${COMMIT}"

echo "Building FontGet (version: $VERSION)"
echo "  Commit: $COMMIT  Date: $DATE"
echo "  Output: $OUTPUT"
echo ""

LDFLAGS="-s -w -X fontget/internal/version.Version=$VERSION -X fontget/internal/version.GitCommit=$COMMIT -X fontget/internal/version.BuildDate=$DATE"
go build -ldflags "$LDFLAGS" -o "$OUTPUT" .

echo ""
echo "Build OK: $OUTPUT"
echo ""
echo "Run:  $OUTPUT version"
echo "      $OUTPUT search roboto"
echo ""

# Show version if we can run the binary
if "$OUTPUT" version 2>/dev/null; then
    :
else
    echo "(Binary could not be run from here; run the commands above in your terminal.)"
fi
