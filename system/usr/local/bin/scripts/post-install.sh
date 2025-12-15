#!/bin/bash
# Post-installation script for RadiantWave
set -e

# Create kiosk group if it doesn't exist
if ! getent group kiosk > /dev/null 2>&1; then
    echo "Creating kiosk group..."
    groupadd --system kiosk
fi

# Create kiosk user if it doesn't exist
if ! getent passwd kiosk > /dev/null 2>&1; then
    echo "Creating kiosk user..."
    useradd --system --gid kiosk --create-home --home-dir /home/kiosk \
        --shell /bin/bash --comment "RadiantWave Kiosk User" kiosk
fi

# Set proper group ownership for writable directories
chgrp -R kiosk /usr/local/share/radiantwave

# Ensure kiosk user can write to the data directory
chmod g+w /usr/local/share/radiantwave

echo "✓ RadiantWave post-install complete"