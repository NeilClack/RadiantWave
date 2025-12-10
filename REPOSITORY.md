# RadiantWave Debian Repository Guide

Quick reference for managing the RadiantWave apt repository with reprepro and Caddy.

## Initial Setup

### 1. Install Dependencies

```bash
sudo apt install reprepro caddy
```

### 2. Create Repository Structure

```bash
# Create repository directories
sudo mkdir -p /srv/radiantwave/apt/{conf,release,dev}
cd /srv/radiantwave/apt
```

### 3. Configure reprepro

Create `/srv/radiantwave/apt/conf/distributions`:

```
Origin: RadiantWave
Label: RadiantWave
Codename: release
Architectures: amd64
Components: main
Description: RadiantWave stable releases
SignWith: no

Origin: RadiantWave
Label: RadiantWave Dev
Codename: dev
Architectures: amd64
Components: main
Description: RadiantWave development builds
SignWith: no
```

### 4. Configure Caddy

Create/edit `/etc/caddy/Caddyfile`:

```
repository.radiantwavetech.com {
    root * /srv/radiantwave/apt
    file_server browse

    # Enable CORS if needed
    header Access-Control-Allow-Origin *

    # Log requests
    log {
        output file /var/log/caddy/radiantwave-repo.log
    }
}
```

Apply Caddy configuration:

```bash
sudo systemctl reload caddy
```

### 5. Set Permissions

```bash
sudo chown -R $USER:www-data /srv/radiantwave/apt
sudo chmod -R 755 /srv/radiantwave/apt
```

## Common Operations

### Adding a New Package

```bash
# For release channel
reprepro -b /srv/radiantwave/apt includedeb release /path/to/radiantwave_2.0.0_amd64.deb

# For dev channel
reprepro -b /srv/radiantwave/apt includedeb dev /path/to/radiantwave-dev_0.0.abc1234_amd64.deb
```

### Updating an Existing Package

```bash
# reprepro automatically replaces if same version
# If newer version, just add it:
reprepro -b /srv/radiantwave/apt includedeb release /path/to/radiantwave_2.1.0_amd64.deb

# To force replace same version:
reprepro -b /srv/radiantwave/apt remove release radiantwave
reprepro -b /srv/radiantwave/apt includedeb release /path/to/radiantwave_2.0.0_amd64.deb
```

### Listing Packages in Repository

```bash
# List all packages in release channel
reprepro -b /srv/radiantwave/apt list release

# List all packages in dev channel
reprepro -b /srv/radiantwave/apt list dev
```

### Removing a Package

```bash
# Remove specific version
reprepro -b /srv/radiantwave/apt remove release radiantwave

# Remove from dev channel
reprepro -b /srv/radiantwave/apt remove dev radiantwave-dev
```

### Rolling Back to Previous Version

```bash
# 1. Remove current version
reprepro -b /srv/radiantwave/apt remove release radiantwave

# 2. Add previous version from backup
reprepro -b /srv/radiantwave/apt includedeb release /backups/radiantwave_1.9.0_amd64.deb
```

**Best Practice:** Always keep backups of previous .deb files in `/srv/radiantwave/backups/`

## Automated Upload Workflow

### Option 1: Manual Upload via SCP

From development machine:

```bash
# Build package
./build.sh release

# Upload to server
scp radiantwave_2.0.0_amd64.deb user@repository.radiantwavetech.com:/tmp/

# SSH to server and add to repository
ssh user@repository.radiantwavetech.com
reprepro -b /srv/radiantwave/apt includedeb release /tmp/radiantwave_2.0.0_amd64.deb
rm /tmp/radiantwave_2.0.0_amd64.deb
```

### Option 2: Upload Script

Create `/srv/radiantwave/bin/add-package.sh` on server:

```bash
#!/bin/bash
set -e

PACKAGE_FILE="$1"
CHANNEL="${2:-release}"

if [ ! -f "$PACKAGE_FILE" ]; then
    echo "Error: Package file not found: $PACKAGE_FILE"
    exit 1
fi

# Backup old package
BACKUP_DIR="/srv/radiantwave/backups"
mkdir -p "$BACKUP_DIR"
PACKAGE_NAME=$(basename "$PACKAGE_FILE")
if [ -f "$BACKUP_DIR/$PACKAGE_NAME" ]; then
    mv "$BACKUP_DIR/$PACKAGE_NAME" "$BACKUP_DIR/${PACKAGE_NAME}.$(date +%Y%m%d-%H%M%S)"
fi

# Add to repository
reprepro -b /srv/radiantwave/apt includedeb "$CHANNEL" "$PACKAGE_FILE"

# Backup new package
cp "$PACKAGE_FILE" "$BACKUP_DIR/"

echo "✓ Added $PACKAGE_NAME to $CHANNEL channel"
```

Usage:
```bash
./add-package.sh /tmp/radiantwave_2.0.0_amd64.deb release
./add-package.sh /tmp/radiantwave-dev_0.0.abc1234_amd64.deb dev
```

## Client Configuration

Users add the repository:

```bash
# Release channel
echo "deb [trusted=yes] https://repository.radiantwavetech.com/release ./" | \
    sudo tee /etc/apt/sources.list.d/radiantwave.list

# Dev channel
echo "deb [trusted=yes] https://repository.radiantwavetech.com/dev ./" | \
    sudo tee /etc/apt/sources.list.d/radiantwave-dev.list

# Install
sudo apt update
sudo apt install radiantwave        # or radiantwave-dev
```

## Repository Maintenance

### Check Repository Integrity

```bash
reprepro -b /srv/radiantwave/apt check release
reprepro -b /srv/radiantwave/apt check dev
```

### Clear Unused Files

```bash
reprepro -b /srv/radiantwave/apt clearvanished
```

### Export Repository Info

```bash
# See what reprepro thinks is in the repository
reprepro -b /srv/radiantwave/apt dumpreferences
```

## Troubleshooting

### Package Won't Update on Client

```bash
# On client machine:
sudo apt update
sudo apt-cache policy radiantwave    # Check available versions
sudo apt install --only-upgrade radiantwave
```

### Repository Rebuild

If repository gets corrupted:

```bash
# Remove all database files
rm -rf /srv/radiantwave/apt/db

# Re-add all packages from backups
reprepro -b /srv/radiantwave/apt includedeb release /srv/radiantwave/backups/radiantwave_*.deb
reprepro -b /srv/radiantwave/apt includedeb dev /srv/radiantwave/backups/radiantwave-dev_*.deb
```

## Directory Structure

```
/srv/radiantwave/
├── apt/
│   ├── conf/
│   │   └── distributions          # reprepro config
│   ├── db/                         # reprepro database (auto-generated)
│   ├── dists/                      # apt metadata (auto-generated)
│   │   ├── release/
│   │   └── dev/
│   ├── pool/                       # actual .deb files (auto-generated)
│   │   └── main/
│   └── incoming/                   # optional: for incoming uploads
├── backups/                        # keep old .deb files here
│   ├── radiantwave_1.9.0_amd64.deb
│   ├── radiantwave_2.0.0_amd64.deb
│   └── ...
└── bin/
    └── add-package.sh              # upload helper script
```

## Quick Reference

```bash
# Add package to release
reprepro -b /srv/radiantwave/apt includedeb release package.deb

# Add package to dev
reprepro -b /srv/radiantwave/apt includedeb dev package.deb

# List packages
reprepro -b /srv/radiantwave/apt list release

# Remove package
reprepro -b /srv/radiantwave/apt remove release radiantwave

# Check repository health
reprepro -b /srv/radiantwave/apt check release
```
