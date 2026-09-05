#!/bin/sh
#
# Record FontGet VHS tapes into vhs/exports/.
#
#   sh scripts/vhs-record.sh          # hero + browse
#   sh scripts/vhs-record.sh hero
#   sh scripts/vhs-record.sh browse
#
# Docker (ghcr.io/charmbracelet/vhs) is preferred, especially on Windows.
# Falls back to a local `vhs` binary (Linux / macOS / WSL).
#
# Inner mode (called inside the VHS container; do not invoke yourself):
#   sh scripts/vhs-record.sh --inner [hero|browse]
#

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

VHS_IMAGE="${FONTGET_VHS_IMAGE:-ghcr.io/charmbracelet/vhs}"
WORK_DIR="$REPO_ROOT/vhs/.work"
BIN_PATH="$WORK_DIR/fontget"
HOME_DIR="$WORK_DIR/home"
EXPORTS_DIR="$REPO_ROOT/vhs/exports"

usage() {
	cat <<'EOF'
Record FontGet VHS demos.

Usage:
  sh scripts/vhs-record.sh          # both tapes
  sh scripts/vhs-record.sh hero
  sh scripts/vhs-record.sh browse

Environment:
  FONTGET_VHS_IMAGE   VHS container image (default: ghcr.io/charmbracelet/vhs)
  FONTGET_VHS_GOARCH  GOARCH for the Linux recorder binary (default: amd64)
EOF
}

# Pre-seed config so recordings never show onboarding, the terms banner,
# or the post-command update prompt (cmd/root.go).
seed_fontget_home() {
	mkdir -p "$HOME/.fontget"

	awk '{sub(/CheckForUpdates: true/, "CheckForUpdates: false")}1' \
		"$REPO_ROOT/internal/config/default_config.yaml" \
		> "$HOME/.fontget/config.yaml"

	state_dir="$HOME/.config/fontget"
	if [ "$(uname -s)" = "Darwin" ]; then
		state_dir="$HOME/Library/Application Support/FontGet"
	fi
	mkdir -p "$state_dir"
	# Stamp sources as fresh so GetRepository skips RunSpinner (which can hang
	# without a TTY). Missing source files still load via GetManifest.
	now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
	cat > "$state_dir/config.json" <<EOF
{
  "first_run_completed": true,
  "agreements_accepted": true,
  "accepted_sources": {},
  "sources_last_updated": "$now"
}
EOF
}

init_fontget_home() {
	# HOME must already point at the throwaway profile.
	unset FONTGET_ACCEPT_AGREEMENTS FONTGET_ACCEPT_DEFAULTS FONTGET_SKIP_ONBOARDING
	seed_fontget_home

	# Download/parse sources so the visible tape is not a first-run spinner.
	echo "Warming FontGet manifest (search roboto)..."
	fontget search roboto >/dev/null
}

run_tapes_inner() {
	tapes="$1"
	export HOME="$HOME_DIR"
	export PATH="$(dirname "$BIN_PATH"):$PATH"
	unset FONTGET_ACCEPT_AGREEMENTS FONTGET_ACCEPT_DEFAULTS FONTGET_SKIP_ONBOARDING

	mkdir -p "$HOME_DIR" "$EXPORTS_DIR"

	echo "Initializing FontGet profile in $HOME_DIR ..."
	init_fontget_home

	for tape in $tapes; do
		tape_file="vhs/tapes/${tape}.tape"
		if [ ! -f "$tape_file" ]; then
			echo "Tape not found: $tape_file" >&2
			exit 1
		fi
		echo "Recording $tape_file ..."
		vhs "$tape_file"
	done

	echo ""
	echo "Exports:"
	ls -lh "$EXPORTS_DIR"
}

if [ "${1:-}" = "--inner" ]; then
	shift
	HOME_DIR="/vhs/.work/home"
	BIN_PATH="/vhs/.work/fontget"
	EXPORTS_DIR="/vhs/exports"
	REPO_ROOT="/vhs"
	cd /vhs

	inner_tapes="hero browse"
	if [ -n "${1:-}" ]; then
		inner_tapes="$1"
	fi
	run_tapes_inner "$inner_tapes"
	exit 0
fi

# --- host ----------------------------------------------------------------------

TAPES="hero browse"
case "${1:-}" in
	-h|--help)
		usage
		exit 0
		;;
	"")
		;;
	hero|browse)
		TAPES="$1"
		;;
	*)
		echo "Unknown tape: $1" >&2
		echo "Use: hero, browse, or omit for both." >&2
		exit 1
		;;
esac

mkdir -p "$WORK_DIR" "$EXPORTS_DIR"
rm -rf "$HOME_DIR"
mkdir -p "$HOME_DIR"

docker_ok() {
	command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

build_linux_binary() {
	arch="${FONTGET_VHS_GOARCH:-amd64}"
	echo "Building Linux ${arch} fontget → $BIN_PATH"
	GOOS=linux GOARCH="$arch" CGO_ENABLED=0 go build -o "$BIN_PATH" .
	chmod +x "$BIN_PATH"
}

docker_mount() {
	# Git Bash converts Unix paths in docker -v; prefer a Windows path when cygpath exists.
	if command -v cygpath >/dev/null 2>&1; then
		cygpath -w "$REPO_ROOT"
	else
		echo "$REPO_ROOT"
	fi
}

if docker_ok; then
	build_linux_binary
	mount="$(docker_mount)"
	echo "Recording with $VHS_IMAGE"
	echo "  mount: $mount -> /vhs"
	# Git Bash otherwise rewrites /vhs in -v and -e HOME.
	export MSYS_NO_PATHCONV=1
	docker run --rm -t \
		-v "${mount}:/vhs" \
		-w /vhs \
		--entrypoint /bin/sh \
		"$VHS_IMAGE" \
		/vhs/scripts/vhs-record.sh --inner $TAPES
	exit 0
fi

if command -v vhs >/dev/null 2>&1; then
	echo "Docker not available; using local vhs"
	os="$(uname -s)"
	case "$os" in
		Linux|Darwin)
			echo "Building native fontget → $BIN_PATH"
			CGO_ENABLED=0 go build -o "$BIN_PATH" .
			chmod +x "$BIN_PATH"
			run_tapes_inner "$TAPES"
			exit 0
			;;
		*)
			echo "Local vhs on $os is not supported for FontGet recordings." >&2
			echo "Install Docker Desktop and re-run, or record from WSL/Linux." >&2
			exit 1
			;;
	esac
fi

echo "Neither Docker nor vhs is available." >&2
echo "Install Docker Desktop (Windows) or VHS + ttyd + ffmpeg (Linux/macOS/WSL)." >&2
exit 1
