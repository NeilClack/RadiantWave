#!/bin/bash
set -euo pipefail

SRC="/home/localuser/.local/share/radiantwave/hyprland.conf"
DEST="/home/localuser/.config/hypr/hyprland.conf"
USER="localuser"
GROUP="localuser"

# Ensure destination directory exists
mkdir -p "$(dirname "$DEST")"

# Copy the config file
cp "$SRC" "$DEST"

# Set ownership and permissions
chown "$USER:$GROUP" "$DEST"
chmod 644 "$DEST"