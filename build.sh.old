#!/bin/bash

# RadiantWave Build Script
#
# Builds the RadiantWave binary and packages it into a Debian package (.deb)

set -euo pipefail

show_help() {
  cat <<EOF
RadiantWave Build Script

Builds the RadiantWave binary and packages it into a Debian package (.deb)

USAGE:
  build.sh [OPTIONS] RELEASE_TYPE

ARGUMENTS:
  RELEASE_TYPE    The release channel
                  Choices: dev | release
                  Required

OPTIONS:
  -h, --help      Show this help message and exit
  --local         Build locally without uploading to repository

EXAMPLES:
  build.sh dev              # Build and upload to dev channel
  build.sh release          # Build and upload to release channel
  build.sh --local dev      # Build locally only (no upload)
EOF
}

# --- Parse args ---
LOCAL_ONLY=false

while [[ $# -gt 0 ]]; do
  case "$1" in
  -h | --help)
    show_help
    exit 0
    ;;
  --local)
    LOCAL_ONLY=true
    shift
    ;;
  *)
    RELEASE_TYPE="$1"
    shift
    ;;
  esac
done

# --- Validate inputs ---
if [[ -z "${RELEASE_TYPE:-}" ]]; then
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

# --- Version / Tag / Package Name ---
if [[ "$RELEASE_TYPE" == "dev" ]]; then
  COMMIT_HASH="$(git rev-parse --short=7 HEAD 2>/dev/null || echo unknown)"
  EPOCH="$(date +%s)"
  # Epoch timestamp ensures versions always increase
  # Format: EPOCH.COMMIT_HASH (e.g., 1733945123.5bb45ec)
  VERSION="${EPOCH}.${COMMIT_HASH}"
  PACKAGE_NAME="radiantwave-dev"
  CONFLICTS="radiantwave"
  CHANNEL="dev"
else
  TAG="$(git describe --tags --abbrev=0 2>/dev/null || true)"
  if [[ -z "$TAG" ]]; then
    echo "Error: No git tag found for release build. Create a tag first with: git tag v2.0.0" >&2
    exit 1
  fi
  # Remove 'v' prefix if present
  VERSION="${TAG#v}"
  PACKAGE_NAME="radiantwave"
  CONFLICTS="radiantwave-dev"
  CHANNEL="release"
fi

# --- Repository settings ---
REPO_USER="nclack"
REPO_HOST="134.122.8.168"
REPO_PATH="/srv/radiantwave/apt"

# --- Prompt confirmation ---
echo "=================================================="
echo " RadiantWave Build Confirmation"
echo "--------------------------------------------------"
echo " Package Name: $PACKAGE_NAME"
echo " Version     : $VERSION"
echo " Channel     : $CHANNEL"
echo " Conflicts   : $CONFLICTS"
if [[ "$LOCAL_ONLY" == true ]]; then
  echo " Mode        : LOCAL ONLY (no upload)"
else
  echo " Mode        : Build and Upload to Repository"
  echo " Repository  : $REPO_USER@$REPO_HOST:$REPO_PATH"
fi
echo "=================================================="
read -rp "Proceed with build? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
  echo "Aborted."
  exit 1
fi
echo
echo "Building $PACKAGE_NAME $VERSION ($CHANNEL channel)"

# --- Clean previous binary ---
BINARY_OUTPUT="./system/usr/local/bin/radiantwave"
rm -f "$BINARY_OUTPUT"

# --- Build binary ---
CGO_ENABLED=1 go build \
  -ldflags="-s -w \
    -X 'radiantwavetech.com/radiantwave/internal/page.GitVersion=${VERSION}'" \
  -o "$BINARY_OUTPUT" radiantwave.go

echo "✓ Built binary: $BINARY_OUTPUT"

# --- Set permissions ---
chmod 755 "$BINARY_OUTPUT"
chmod 755 ./system/usr/local/bin/radiantwave-updater.py
chmod 755 ./system/usr/local/bin/scripts/post-install.sh
chmod 644 ./system/etc/polkit-1/rules.d/99-radiantwave-updater.rules

# Set asset permissions
find ./system/usr/local/share/radiantwave -type d -exec chmod 755 {} \;
find ./system/usr/local/share/radiantwave -type f -exec chmod 644 {} \;

echo "✓ Set file permissions"

# --- Create Debian package ---
PACKAGE_DIR="./pkg/${PACKAGE_NAME}_${VERSION}_amd64"
rm -rf ./pkg
mkdir -p "$PACKAGE_DIR/DEBIAN"

echo "✓ Created package directory: $PACKAGE_DIR"

# Copy system files to package directory
cp -r ./system/* "$PACKAGE_DIR/"

echo "✓ Copied system files to package"

# --- Template radiantwave-updater.py (in package, not source) ---
UPDATER_PKG_PATH="$PACKAGE_DIR/usr/local/bin/radiantwave-updater.py"
if [[ -f "$UPDATER_PKG_PATH" ]]; then
  sed -i -e "s|__CHANNEL__|${PACKAGE_NAME}|g" "$UPDATER_PKG_PATH"
  echo "✓ Templated updater script with package name: $PACKAGE_NAME"
else
  echo "ERROR: $UPDATER_PKG_PATH not found" >&2
  exit 1
fi

# Create control file from template
sed -e "s/__PACKAGE_NAME__/${PACKAGE_NAME}/g" \
  -e "s/__VERSION__/${VERSION}/g" \
  -e "s/__CONFLICTS__/${CONFLICTS}/g" \
  -e "s/__CHANNEL__/${CHANNEL}/g" \
  debian/control.template >"$PACKAGE_DIR/DEBIAN/control"

echo "✓ Created DEBIAN/control"

# Copy package maintainer scripts
cp debian/preinst "$PACKAGE_DIR/DEBIAN/preinst"
cp debian/postinst "$PACKAGE_DIR/DEBIAN/postinst"
cp debian/prerm "$PACKAGE_DIR/DEBIAN/prerm"
chmod 755 "$PACKAGE_DIR/DEBIAN/preinst"
chmod 755 "$PACKAGE_DIR/DEBIAN/postinst"
chmod 755 "$PACKAGE_DIR/DEBIAN/prerm"

echo "✓ Copied DEBIAN scripts (preinst, postinst, prerm)"

# Build the .deb package
DEB_FILE="${PACKAGE_NAME}_${VERSION}_amd64.deb"
dpkg-deb --build "$PACKAGE_DIR" "$DEB_FILE"

echo "✓ Built Debian package: $DEB_FILE"

# Set proper permissions on the .deb file (readable by all, including _apt user)
chmod 644 "$DEB_FILE"

echo "✓ Set package permissions (644)"

# Generate SHA256 checksum
sha256sum "$DEB_FILE" >"${DEB_FILE}.sha256"
chmod 644 "${DEB_FILE}.sha256"

echo "✓ Generated SHA256 checksum"

# Clean up staging directory
rm -rf ./pkg

# --- Upload to repository (if not local-only) ---
if [[ "$LOCAL_ONLY" == false ]]; then
  echo ""
  echo "Uploading to repository..."

  # Upload .deb to server
  if ! scp "$DEB_FILE" "${REPO_USER}@${REPO_HOST}:/tmp/"; then
    echo "ERROR: Failed to upload package to server" >&2
    exit 1
  fi

  echo "✓ Uploaded $DEB_FILE to server"

  # Add to repository using reprepro
  echo "Adding package to $CHANNEL repository..."
  if ! ssh "${REPO_USER}@${REPO_HOST}" \
    "reprepro -b $REPO_PATH includedeb $CHANNEL /tmp/$DEB_FILE && rm /tmp/$DEB_FILE"; then
    echo "ERROR: Failed to add package to repository" >&2
    exit 1
  fi

  echo "✓ Added to repository"

  echo ""
  echo "=================================================="
  echo " Build & Upload Complete!"
  echo "--------------------------------------------------"
  echo " Package     : $DEB_FILE"
  echo " Version     : $VERSION"
  echo " Channel     : $CHANNEL"
  echo " Repository  : $REPO_USER@$REPO_HOST"
  echo "=================================================="
  echo ""
  echo "Installation on client systems:"
  echo "  sudo apt update"
  echo "  sudo apt install $PACKAGE_NAME"
  echo ""
  echo "Or upgrade existing installation:"
  echo "  sudo apt update"
  echo "  sudo apt upgrade $PACKAGE_NAME"
else
  echo ""
  echo "=================================================="
  echo " Build Complete (Local Only)!"
  echo "--------------------------------------------------"
  echo " Package     : $DEB_FILE"
  echo " Version     : $VERSION"
  echo " Channel     : $CHANNEL"
  echo " Location    : $(pwd)/$DEB_FILE"
  echo "=================================================="
  echo ""
  echo "Installation commands:"
  echo "  sudo dpkg -i $DEB_FILE"
  echo "  sudo apt install -f  # Fix any dependency issues"
fi

