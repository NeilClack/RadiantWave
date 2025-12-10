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

EXAMPLES:
  build.sh dev         # Build radiantwave-dev package
  build.sh release     # Build radiantwave package
EOF
}

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
  -h | --help)
    show_help
    exit 0
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
  VERSION="${COMMIT_HASH}"
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

# --- Prompt confirmation ---
echo "=================================================="
echo " RadiantWave Build Confirmation"
echo "--------------------------------------------------"
echo " Package Name: $PACKAGE_NAME"
echo " Version     : $VERSION"
echo " Channel     : $CHANNEL"
echo " Conflicts   : $CONFLICTS"
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
chmod 755 ./system/usr/local/bin/scripts/radiantwave-installer.sh
chmod 644 ./system/etc/polkit-1/rules.d/99-radiantwave-updater.rules

# Set asset permissions
find ./system/usr/local/share/radiantwave -type d -exec chmod 755 {} \;
find ./system/usr/local/share/radiantwave -type f -exec chmod 644 {} \;

echo "✓ Set file permissions"

# --- Template radiantwave-updater.py ---
UPDATER_PATH="./system/usr/local/bin/radiantwave-updater.py"
if [[ -f "$UPDATER_PATH" ]]; then
  sed -i -e "s|__CHANNEL__|${PACKAGE_NAME}|g" "$UPDATER_PATH"
  echo "✓ Templated updater script with package name: $PACKAGE_NAME"
else
  echo "ERROR: $UPDATER_PATH not found" >&2
  exit 1
fi

# --- VERSION files ---
echo "${VERSION}" > ./system/usr/local/share/radiantwave/VERSION
echo "${VERSION}" > ./VERSION

echo "✓ Created VERSION files"

# --- Create Debian package ---
PACKAGE_DIR="./pkg/${PACKAGE_NAME}_${VERSION}_amd64"
rm -rf ./pkg
mkdir -p "$PACKAGE_DIR/DEBIAN"

echo "✓ Created package directory: $PACKAGE_DIR"

# Copy system files to package directory
cp -r ./system/* "$PACKAGE_DIR/"

echo "✓ Copied system files to package"

# Create control file from template
sed -e "s/__PACKAGE_NAME__/${PACKAGE_NAME}/g" \
    -e "s/__VERSION__/${VERSION}/g" \
    -e "s/__CONFLICTS__/${CONFLICTS}/g" \
    -e "s/__CHANNEL__/${CHANNEL}/g" \
    debian/control.template > "$PACKAGE_DIR/DEBIAN/control"

echo "✓ Created DEBIAN/control"

# Copy postinst and prerm scripts
cp debian/postinst "$PACKAGE_DIR/DEBIAN/postinst"
cp debian/prerm "$PACKAGE_DIR/DEBIAN/prerm"
chmod 755 "$PACKAGE_DIR/DEBIAN/postinst"
chmod 755 "$PACKAGE_DIR/DEBIAN/prerm"

echo "✓ Copied DEBIAN scripts"

# Build the .deb package
DEB_FILE="${PACKAGE_NAME}_${VERSION}_amd64.deb"
dpkg-deb --build "$PACKAGE_DIR" "$DEB_FILE"

echo "✓ Built Debian package: $DEB_FILE"

# Generate SHA256 checksum
sha256sum "$DEB_FILE" > "${DEB_FILE}.sha256"

echo "✓ Generated SHA256 checksum"

# Clean up staging directory
rm -rf ./pkg

echo ""
echo "=================================================="
echo " Build Complete!"
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
echo ""
echo "Or add to apt repository and install with:"
echo "  sudo apt update"
echo "  sudo apt install $PACKAGE_NAME"