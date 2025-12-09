#!/bin/bash
# radiantwave-install-helper.sh
# Helper script run via pkexec to install RadiantWave updates
# This runs with elevated privileges to extract to system directories

set -euo pipefail

TARBALL="$1"

if [[ ! -f "$TARBALL" ]]; then
    echo "ERROR: Tarball not found: $TARBALL" >&2
    exit 1
fi

# Extract to root filesystem
echo "Extracting $TARBALL..."
tar --no-same-owner -xJf "$TARBALL" -C /

# Set executable permissions
echo "Setting permissions..."
chmod +x /usr/local/bin/radiantwave 2>/dev/null || true
chmod +x /usr/local/bin/radiantwave-updater.py 2>/dev/null || true

if [[ -d /usr/local/bin/scripts ]]; then
    chmod +x /usr/local/bin/scripts/*.sh 2>/dev/null || true
fi

echo "Installation complete"
exit 0