#!/bin/bash
# Run after extracting RadiantWave tarball
set -e

# Set proper ownership for writable directories
chgrp -R kiosk /usr/local/share/radiantwave

echo "✓ RadiantWave post-install complete"