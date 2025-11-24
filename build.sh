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
  build.sh [SYSTEM_TYPE] [RELEASE_TYPE]

ARGUMENTS:
  SYSTEM_TYPE     The target system type
                  Choices: home | commercial
                  Default: home

  RELEASE_TYPE    The release channel
                  Choices: dev | beta | release
                  Default: (auto from Git branch: main->release, dev->dev, beta->beta; otherwise dev)

OPTIONS:
  -h, --help      Show this help message and exit
EOF
}

# --- Parse args ---
if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  show_help
  exit 0
fi

SYSTEM_TYPE="${1:-home}"

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

# Assert it landed
REQ="$PKGROOT/usr/local/share/radiantwave/hyprland.conf"
[[ -f "$REQ" ]] || {
  echo "ERROR: Missing $REQ after copy"
  exit 1
}
chmod 644 "$REQ"

# Ensure binaries are executable even if git perms are off locally
if [[ -d "$PKGROOT/usr/local/bin" ]]; then
  find "$PKGROOT/usr/local/bin" -type f -exec chmod 755 {} \;
fi

# --- Drop compiled binary ---
install -Dm755 ./radiantwave "$PKGROOT/usr/local/bin/radiantwave"

# --- Template radiantwave-updater in-place (if present from ./system) ---
UPDATER_PATH="$PKGROOT/usr/local/bin/radiantwave-updater"
if [[ -f "$UPDATER_PATH" ]]; then
  sed -i -e "s|__CHANNEL__|${RELEASE_TYPE}|g" \
    -e "s|__SYSTEM_TYPE__|${SYSTEM_TYPE}|g" "$UPDATER_PATH"
  chmod 755 "$UPDATER_PATH"
else
  echo "WARN: $UPDATER_PATH not found; skipping templating."
fi

# --- Assets ---
install -d -m755 "$PKGROOT/usr/local/share/radiantwave"
cp -a ./assets/. "$PKGROOT/usr/local/share/radiantwave/"
find "$PKGROOT/usr/local/share/radiantwave" -type d -exec chmod 755 {} \;
find "$PKGROOT/usr/local/share/radiantwave" -type f -exec chmod 644 {} \;

# --- VERSION files ---
echo "${TAG}" >"$PKGROOT/usr/local/share/radiantwave/VERSION"
echo "${TAG}" >"./VERSION"

# --- Create artifact ---
OUT="radiantwave-${SYSTEM_TYPE}-${TAG}.tar.xz"
tar --numeric-owner -C "$PKGROOT" -cJf "$OUT" .
sha256sum "$OUT" >"${OUT}.sha256"

echo "Built: $OUT"

# --- Upload ---
# rsync -av --rsync-path="sudo rsync" --chown=www-data:www-data --progress \
#   "$OUT" "${OUT}.sha256" VERSION \
#   "$REMOTE_USER@$REMOTE_HOST:$REMOTE_LOCATION/"

# ssh "$REMOTE_USER@$REMOTE_HOST" "ls -l $REMOTE_LOCATION"

# echo "Uploaded to $REMOTE_HOST:$REMOTE_LOCATION"
# echo "Cleaning up local directory"
# rm -rf "$OUT" "$OUT.sha256" "VERSION" "$PKGROOT" radiantwave
