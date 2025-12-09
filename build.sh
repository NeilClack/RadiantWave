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
  build.sh [OPTIONS] RELEASE_TYPE

ARGUMENTS:
  RELEASE_TYPE    The release channel
                  Choices: dev | release
                  Required

OPTIONS:
  -h, --help      Show this help message and exit

EXAMPLES:
  build.sh dev         # Build for dev channel
  build.sh release     # Build for release channel
EOF
}

# --- Parse args ---
if [[ $# -eq 0 || "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  show_help
  exit 0
fi

RELEASE_TYPE="${1:-}"

# --- Validate inputs ---
if [[ -z "$RELEASE_TYPE" ]]; then
  echo "Error: RELEASE_TYPE is required" >&2
  show_help
  exit 1
fi

case "$RELEASE_TYPE" in
dev | release) ;;
*)
  echo "Error: RELEASE_TYPE must be 'dev' or 'release' (got: $RELEASE_TYPE)" >&2
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
    echo "Error: No git tag found for release build. Create a tag first with: git tag v0.1.2" >&2
    exit 1
  fi
fi

# --- Prompt confirmation ---
echo "=================================================="
echo " RadiantWave Build Confirmation"
echo "--------------------------------------------------"
echo " Release Type: $RELEASE_TYPE"
echo " Version     : $TAG"
echo "=================================================="
read -rp "Proceed with build using this version? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 1
fi
echo
echo "Building RadiantWave $TAG ($RELEASE_TYPE)"

# --- Remote settings ---
REMOTE_USER=$USER
REMOTE_HOST="basic_fileserver"
REMOTE_LOCATION="/srv/radiantwave/basic/$RELEASE_TYPE"

# --- Clean previous binary ---
BINARY_OUTPUT="./system/usr/local/bin/radiantwave"
rm -f "$BINARY_OUTPUT"

# --- Build binary ---
CGO_ENABLED=1 go build \
  -ldflags="-s -w \
    -X 'radiantwavetech.com/radiantwave/internal/page.GitVersion=${TAG}'" \
  -o "$BINARY_OUTPUT" radiantwave.go

echo "✓ Built binary: $BINARY_OUTPUT"

# --- Set permissions ---
chmod 755 "$BINARY_OUTPUT"
chmod 755 ./system/usr/local/bin/radiantwave-updater.py
chmod 755 ./system/usr/local/bin/scripts/radiantwave-install-helper.sh
chmod 644 ./system/etc/polkit-1/rules.d/99-radiantwave-updater.rules

# Set asset permissions
find ./system/usr/local/share/radiantwave -type d -exec chmod 755 {} \;
find ./system/usr/local/share/radiantwave -type f -exec chmod 644 {} \;

echo "✓ Set file permissions"

# --- Template radiantwave-updater.py ---
UPDATER_PATH="./system/usr/local/bin/radiantwave-updater.py"
if [[ -f "$UPDATER_PATH" ]]; then
  sed -i -e "s|__CHANNEL__|${RELEASE_TYPE}|g" "$UPDATER_PATH"
  echo "✓ Templated updater script"
else
  echo "ERROR: $UPDATER_PATH not found" >&2
  exit 1
fi

# --- VERSION files ---
echo "${TAG}" > ./system/usr/local/share/radiantwave/VERSION
echo "${TAG}" > ./VERSION

echo "✓ Created VERSION files"

# --- Create tarball ---
OUT="radiantwave-${TAG}.tar.xz"
tar --numeric-owner -C ./system -cJf "$OUT" .
sha256sum "$OUT" > "${OUT}.sha256"

echo "✓ Created tarball: $OUT"

# --- Upload ---
echo ""
echo "Uploading to $REMOTE_HOST:$REMOTE_LOCATION ..."

rsync -av --rsync-path="sudo rsync" --chown=www-data:www-data --progress \
  "$OUT" "${OUT}.sha256" VERSION \
  "$REMOTE_USER@$REMOTE_HOST:$REMOTE_LOCATION/"

ssh "$REMOTE_USER@$REMOTE_HOST" "ls -lh $REMOTE_LOCATION"

echo ""
echo "=================================================="
echo " Build Complete!"
echo "--------------------------------------------------"
echo " Version     : $TAG"
echo " Tarball     : $OUT"
echo " Remote Path : $REMOTE_HOST:$REMOTE_LOCATION"
echo "=================================================="
echo ""
echo "Installation command (on target system):"
echo "  sudo tar --no-same-owner -xJf $OUT -C /"