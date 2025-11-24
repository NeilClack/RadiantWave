#!/bin/bash

# RadiantWave Build & Deployment Script
#
# Builds the RadiantWave binary, packages it into a tarball, and uploads it
# to the designated file server with correct ownership and permissions.

set -euo pipefail

show_help() {
  cat <<EOF
RadiantWave Build & Deployment Script

Builds the RadiantWave binary, packages it into a tarball, and uploads it
to the designated file server with correct ownership and permissions.

USAGE:
  build.sh [OPTIONS] [SYSTEM_TYPE] [RELEASE_TYPE]

ARGUMENTS:
  SYSTEM_TYPE     The target system type
                  Choices: home | commercial
                  Default: home

  RELEASE_TYPE    The release channel
                  Choices: dev | beta | release
                  Default: (auto from Git branch: main->release, dev->dev, beta->beta; otherwise dev)

OPTIONS:
  -h, --help      Show this help message and exit
  --release       Build for release (uses 'localuser' in paths)
                  Default is local dev build (uses current \$USER)
EOF
}

# --- Parse args ---
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  show_help
  exit 0
fi

# Check for --release flag
RELEASE_BUILD=false
POSITIONAL_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --release)
      RELEASE_BUILD=true
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      POSITIONAL_ARGS+=("$1")
      shift
      ;;
  esac
done

# Set positional arguments back
set -- "${POSITIONAL_ARGS[@]:-}"

SYSTEM_TYPE="${1:-home}"

# Determine target username for paths
if [[ "$RELEASE_BUILD" == true ]]; then
  TARGET_USER="localuser"
  BUILD_MODE="release"
else
  TARGET_USER="$USER"
  BUILD_MODE="local"
fi

detect_branch() {
  if [[ -n "${GITHUB_REF_NAME:-}" ]]; then
    echo "$GITHUB_REF_NAME"
    return
  fi
  if [[ -n "${CI_COMMIT_BRANCH:-}" ]]; then
    echo "$CI_COMMIT_BRANCH"
    return
  fi
  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"
}

map_branch_to_channel() {
  local b="$1"
  case "$b" in
  main | master) echo "release" ;;
  dev) echo "dev" ;;
  beta) echo "beta" ;;
  *) echo "dev" ;;
  esac
}

BRANCH="$(detect_branch)"
AUTO_CHANNEL="$(map_branch_to_channel "$BRANCH")"
RELEASE_TYPE="${2:-$AUTO_CHANNEL}"

# --- Validate inputs ---
case "$SYSTEM_TYPE" in
home | commercial) ;;
*)
  echo "Error: SYSTEM_TYPE must be 'home' or 'commercial' (got: $SYSTEM_TYPE)" >&2
  exit 1
  ;;
esac

case "$RELEASE_TYPE" in
dev | beta | release) ;;
*)
  echo "Error: RELEASE_TYPE must be 'dev', 'beta', or 'release' (got: $RELEASE_TYPE)" >&2
  exit 1
  ;;
esac

# --- Version / Tag ---
if [[ "$RELEASE_TYPE" == "dev" ]]; then
  COMMIT_HASH="$(git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)"
  TAG="${COMMIT_HASH}"
else
  TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  if [[ -z "$TAG" ]]; then
    TAG="v$(date +%Y%m%d).$(git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)"
  fi
fi

# --- Prompt confirmation (AFTER version is known) ---
echo "=================================================="
echo " RadiantWave Build Confirmation"
echo "--------------------------------------------------"
echo " System Type : $SYSTEM_TYPE"
echo " Release Type: $RELEASE_TYPE (branch: $BRANCH)"
echo " Build Mode  : $BUILD_MODE (target user: $TARGET_USER)"
echo " Version     : $TAG"
echo "=================================================="
read -rp "Proceed with build using this version? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 1
fi
echo
echo "Building RadiantWave $TAG ($SYSTEM_TYPE / $RELEASE_TYPE)"

# --- Remote settings ---
REMOTE_USER=$USER
REMOTE_HOST="basic_fileserver"
REMOTE_LOCATION="/srv/radiantwave/basic/$RELEASE_TYPE/$SYSTEM_TYPE"

# --- Build binary ---
CGO_ENABLED=1 go build -trimpath -buildmode=pie \
  -ldflags="-s -w \
    -X 'radiantwavetech.com/radiantwave/internal/page.GitVersion=${TAG}' \
    -X 'radiantwavetech.com/radiantwave/internal/config.SystemType=${SYSTEM_TYPE}'" \
  -o radiantwave radiantwave.go

# --- Package root ---
PKGROOT="$(pwd)/pkgroot"
rm -rf "$PKGROOT"
mkdir -p "$PKGROOT"

# --- Stage everything from ./system into the pkg root ---
# This will place:
#   - /usr/local/bin/* (e.g., kiosk-session, radiantwave-updater [templated below])
#   - /home/localuser/.config/hypr/hyprland.conf
# and any other staged paths you keep under ./system/
# --- Stage everything from ./system into the pkg root (with dotfiles) ---
mkdir -p "$PKGROOT"
cp -a ./system/. "$PKGROOT/"

# Rename source username (nclack) to target username if different
SRC_USER="nclack"
if [[ "$TARGET_USER" != "$SRC_USER" && -d "$PKGROOT/home/$SRC_USER" ]]; then
  mv "$PKGROOT/home/$SRC_USER" "$PKGROOT/home/$TARGET_USER"
fi

# Define user home for convenience
USER_HOME="$PKGROOT/home/$TARGET_USER"

# Assert it landed
REQ="$USER_HOME/.local/share/radiantwave/hyprland.conf"
[[ -f "$REQ" ]] || {
  echo "ERROR: Missing $REQ after copy"
  exit 1
}
chmod 644 "$REQ"

# Ensure binaries are executable even if git perms are off locally
if [[ -d "$USER_HOME/.local/" ]]; then
  find "$USER_HOME/.local/bin" -type f -exec chmod 755 {} \;
fi

# --- Drop compiled binary ---
install -Dm755 ./radiantwave "$USER_HOME/.local/bin/radiantwave"

# --- Template radiantwave-updater in-place (if present from ./system) ---
UPDATER_PATH="$USER_HOME/.local/bin/radiantwave-updater"
if [[ -f "$UPDATER_PATH" ]]; then
  sed -i -e "s|__CHANNEL__|${RELEASE_TYPE}|g" \
    -e "s|__SYSTEM_TYPE__|${SYSTEM_TYPE}|g" "$UPDATER_PATH"
  chmod 755 "$UPDATER_PATH"
else
  echo "WARN: $UPDATER_PATH not found; skipping templating."
fi

# --- Assets ---
install -d -m755 "$USER_HOME/.local/share/radiantwave"
cp -a ./assets/. "$USER_HOME/.local/share/radiantwave/"
find "$USER_HOME/.local/share/radiantwave" -type d -exec chmod 755 {} \;
find "$USER_HOME/.local/share/radiantwave" -type f -exec chmod 644 {} \;

# --- VERSION files ---
echo "${TAG}" >"$USER_HOME/.local/share/radiantwave/VERSION"
echo "${TAG}" >"./VERSION"

# --- Create artifact ---
if [[ "$RELEASE_BUILD" == true ]]; then
  # Release build: absolute paths from root (for deployment to /home/localuser)
  OUT="radiantwave-${SYSTEM_TYPE}-${TAG}.tar.xz"
  tar --numeric-owner -C "$PKGROOT" -cJf "$OUT" .
  sha256sum "$OUT" >"${OUT}.sha256"
  echo "Built: $OUT"
  echo ""
  echo "Release installation:"
  echo "  sudo tar --no-same-owner -xJvf $OUT -C /"
else
  # Local dev build: relative paths from user home (extracts to \$HOME)
  OUT="radiantwave-${SYSTEM_TYPE}-${TAG}.tar.xz"
  tar --numeric-owner -C "$USER_HOME" -cJf "$OUT" .
  sha256sum "$OUT" >"${OUT}.sha256"
  echo "Built: $OUT"
  echo ""
  echo "Local development installation (no sudo needed):"
  echo "  tar --no-same-owner -xJvf $OUT -C \$HOME"
fi

# --- Upload ---
# rsync -av --rsync-path="sudo rsync" --chown=www-data:www-data --progress \
#   "$OUT" "${OUT}.sha256" VERSION \
#   "$REMOTE_USER@$REMOTE_HOST:$REMOTE_LOCATION/"

# ssh "$REMOTE_USER@$REMOTE_HOST" "ls -l $REMOTE_LOCATION"

# echo "Uploaded to $REMOTE_HOST:$REMOTE_LOCATION"
# echo "Cleaning up local directory"
# rm -rf "$OUT" "$OUT.sha256" "VERSION" "$PKGROOT" radiantwave
