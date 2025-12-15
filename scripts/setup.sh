#!/bin/bash
set -e

# RadiantWave Repository Setup Script
# Usage: curl -fsSL https://repository.radiantwavetech.com/setup.sh | sudo bash

echo "=== RadiantWave Repository Setup ==="

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Error: This script must be run as root"
  echo "Please run: curl -fsSL https://repository.radiantwavetech.com/setup.sh | sudo bash"
  exit 1
fi

# Check if running on Ubuntu/Debian
if [ ! -f /etc/debian_version ]; then
  echo "Error: This script is for Ubuntu/Debian systems only"
  exit 1
fi

REPO_URL="https://repository.radiantwavetech.com"
REPO_NAME="radiantwave"
SOURCES_LIST="/etc/apt/sources.list.d/${REPO_NAME}.list"
KEYRING_FILE="/usr/share/keyrings/${REPO_NAME}-archive-keyring.gpg"

echo "Adding RadiantWave repository..."

# Add the repository to sources.list.d
# echo "deb [signed-by=${KEYRING_FILE}] ${REPO_URL} stable main" >"${SOURCES_LIST}"

echo "deb [trusted=yes] ${REPO_URL} release main" >"${SOURCES_LIST}"

# Download and install the GPG key (you'll need to provide this)
# Uncomment and modify once you have a GPG key:
# curl -fsSL "${REPO_URL}/KEY.gpg" | gpg --dearmor -o "${KEYRING_FILE}"

echo "Updating package lists..."
apt update

echo "Upgrading system packages..."
apt upgrade -y

echo "Installing Tailscale..."
if ! command -v tailscale &> /dev/null; then
    curl -fsSL https://tailscale.com/install.sh | sh
else
    echo "✓ Tailscale already installed"
fi

echo "Setting Tailscale operator..."
if tailscale set --operator=kiosk; then
    echo "✓ Tailscale operator set to kiosk"
else
    echo "⚠ Warning: Could not set Tailscale operator"
fi

echo "Installing RadiantWave..."
apt install -y radiantwave

echo ""
echo "=== Installation Complete ==="
echo "RadiantWave has been installed successfully."
echo ""