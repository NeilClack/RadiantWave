#!/bin/bash
# 00-migrate-v0.1.2.sh
#
# Post-install migration script for RadiantWave v0.1.2
# Called automatically by the OLD updater (v0.1.0/v0.1.1-r1) after extracting v0.1.2
#
# This script:
# - Migrates data from /usr/local/ to /home/localuser/.local/
# - Updates hyprland.conf
# - Sets up user crontab with @reboot updater
# - Cleans up old system files including itself

set -euo pipefail

SCRIPT="$(basename "$0")"
log() { echo "$*"; logger -t "$SCRIPT" "$*"; }

# --- Configuration ---
TARGET_USER="localuser"
MIGRATION_MARKER="/home/${TARGET_USER}/.local/share/radiantwave/.migration-v0.1.2-complete"

# Old system paths
OLD_SHARE_DIR="/usr/local/share/radiantwave"

# New system paths
NEW_HOME="/home/${TARGET_USER}"
NEW_SHARE_DIR="${NEW_HOME}/.local/share/radiantwave"
NEW_BIN_DIR="${NEW_HOME}/.local/bin"
NEW_UPDATER="${NEW_BIN_DIR}/radiantwave-updater"

# Hyprland config
HYPRLAND_CONFIG_DIR="${NEW_HOME}/.config/hypr"
HYPRLAND_CONFIG="${HYPRLAND_CONFIG_DIR}/hyprland.conf"
NEW_HYPRLAND_SOURCE="${NEW_SHARE_DIR}/hyprland.conf"

# Crontab
SYSTEM_CRONTAB="/etc/crontab"

log "========================================="
log "RadiantWave v0.1.2 Migration"
log "========================================="

# Check if already migrated
if [[ -f "$MIGRATION_MARKER" ]]; then
    log "Migration already complete. Skipping."
    exit 0
fi

# Verify new system files are present (should have been extracted by old updater)
if [[ ! -f "$NEW_UPDATER" ]]; then
    log "ERROR: New updater not found at ${NEW_UPDATER}"
    log "This script expects v0.1.2 files to already be extracted."
    exit 1
fi

# --- Step 1: Migrate data from old to new locations ---
log ""
log "Step 1: Migrating data..."

if [[ -d "$OLD_SHARE_DIR" ]]; then
    log "  Found old data directory at ${OLD_SHARE_DIR}"

    # Ensure new directory exists
    mkdir -p "$NEW_SHARE_DIR"

    # Migrate database (preserve user data)
    OLD_DB="${OLD_SHARE_DIR}/data.db"
    NEW_DB="${NEW_SHARE_DIR}/data.db"

    if [[ -f "$OLD_DB" ]]; then
        if [[ -f "$NEW_DB" ]]; then
            # Both exist - keep newer
            if [[ "$OLD_DB" -nt "$NEW_DB" ]]; then
                log "  Migrating newer database from old location"
                cp -a "$OLD_DB" "$NEW_DB"
            else
                log "  Keeping newer database in new location"
            fi
        else
            log "  Migrating database"
            cp -a "$OLD_DB" "$NEW_DB"
        fi
    fi

    # Migrate VERSION file
    if [[ -f "${OLD_SHARE_DIR}/VERSION" ]]; then
        log "  Migrating VERSION file"
        cp -a "${OLD_SHARE_DIR}/VERSION" "${NEW_SHARE_DIR}/VERSION"
    fi

    # Migrate .sha256 checksum files
    for shafile in "${OLD_SHARE_DIR}"/*.sha256 2>/dev/null; do
        if [[ -f "$shafile" ]]; then
            filename="$(basename "$shafile")"
            log "  Migrating checksum: ${filename}"
            cp -a "$shafile" "${NEW_SHARE_DIR}/${filename}"
        fi
    done

    # Fix ownership
    chown -R "${TARGET_USER}:${TARGET_USER}" "$NEW_SHARE_DIR"
    log "  ✓ Data migration complete"
else
    log "  No old data directory found"
fi

# --- Step 2: Update hyprland.conf ---
log ""
log "Step 2: Updating hyprland configuration..."

if [[ -f "$NEW_HYPRLAND_SOURCE" ]]; then
    mkdir -p "$HYPRLAND_CONFIG_DIR"

    # Backup existing config if present
    if [[ -f "$HYPRLAND_CONFIG" ]]; then
        BACKUP="${HYPRLAND_CONFIG}.backup-$(date +%Y%m%d-%H%M%S)"
        log "  Backing up: ${BACKUP}"
        cp "$HYPRLAND_CONFIG" "$BACKUP"
    fi

    log "  Installing hyprland.conf"
    cp "$NEW_HYPRLAND_SOURCE" "$HYPRLAND_CONFIG"
    chown "${TARGET_USER}:${TARGET_USER}" "$HYPRLAND_CONFIG"
    chmod 644 "$HYPRLAND_CONFIG"
    log "  ✓ Hyprland config updated"
else
    log "  ⚠ Warning: hyprland.conf not found at ${NEW_HYPRLAND_SOURCE}"
fi

# --- Step 3: Update crontab configuration ---
log ""
log "Step 3: Updating crontab..."

# Remove old system crontab entry
if [[ -f "$SYSTEM_CRONTAB" ]]; then
    if grep -q "radiantwave-updater" "$SYSTEM_CRONTAB"; then
        log "  Removing old crontab entries from ${SYSTEM_CRONTAB}"
        cp "$SYSTEM_CRONTAB" "${SYSTEM_CRONTAB}.backup-$(date +%Y%m%d-%H%M%S)"
        sed -i '/radiantwave-updater/d' "$SYSTEM_CRONTAB"
        log "  ✓ Removed old crontab entries"
    fi
fi

# Set up user crontab for localuser
log "  Setting up user crontab for ${TARGET_USER}..."

TEMP_CRON=$(mktemp)
if crontab -u "${TARGET_USER}" -l &>/dev/null; then
    crontab -u "${TARGET_USER}" -l > "$TEMP_CRON"
    # Remove any existing radiantwave-updater entries
    sed -i '/radiantwave-updater/d' "$TEMP_CRON"
fi

# Add @reboot entry for the new updater
echo "@reboot ${NEW_UPDATER}" >> "$TEMP_CRON"

# Install the new crontab
crontab -u "${TARGET_USER}" "$TEMP_CRON"
rm -f "$TEMP_CRON"

log "  ✓ User crontab configured: @reboot ${NEW_UPDATER}"

# --- Step 4: Set executable permissions ---
log ""
log "Step 4: Setting permissions..."

if [[ -d "${NEW_BIN_DIR}" ]]; then
    find "${NEW_BIN_DIR}" -type f -name "*.sh" -exec chmod +x {} \;
    find "${NEW_BIN_DIR}" -type f -name "radiantwave*" -exec chmod +x {} \;
    log "  ✓ Permissions set"
fi

# --- Step 5: Create migration marker ---
log ""
log "Step 5: Creating migration marker..."

mkdir -p "$(dirname "$MIGRATION_MARKER")"
{
    echo "Migration completed: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Script: $SCRIPT"
} > "$MIGRATION_MARKER"
chown "${TARGET_USER}:${TARGET_USER}" "$MIGRATION_MARKER"

log "  ✓ Migration marker created"

# --- Step 6: Clean up old system files ---
log ""
log "Step 6: Cleaning up old system..."

REMOVED_ANYTHING=false

# Remove old binaries
if [[ -f /usr/local/bin/radiantwave ]]; then
    log "  Removing /usr/local/bin/radiantwave"
    rm -f /usr/local/bin/radiantwave
    REMOVED_ANYTHING=true
fi

if [[ -f /usr/local/bin/radiantwave-updater ]]; then
    log "  Removing /usr/local/bin/radiantwave-updater"
    rm -f /usr/local/bin/radiantwave-updater
    REMOVED_ANYTHING=true
fi

# Remove standalone migration script (manual testing version)
if [[ -f /usr/local/bin/radiantwave-migrate-v0.1.2.sh ]]; then
    log "  Removing /usr/local/bin/radiantwave-migrate-v0.1.2.sh"
    rm -f /usr/local/bin/radiantwave-migrate-v0.1.2.sh
    REMOVED_ANYTHING=true
fi

# Remove old share directory
if [[ -d "$OLD_SHARE_DIR" ]]; then
    log "  Removing ${OLD_SHARE_DIR}"
    rm -rf "$OLD_SHARE_DIR"
    REMOVED_ANYTHING=true
fi

# Remove intermediate directory if exists
if [[ -d /usr/local/bin/radiantwave ]]; then
    log "  Removing /usr/local/bin/radiantwave directory"
    rm -rf /usr/local/bin/radiantwave
    REMOVED_ANYTHING=true
fi

# Remove old scripts directory (including this script!)
# This runs last so this script can complete first
if [[ -d /usr/local/bin/scripts ]]; then
    log "  Removing /usr/local/bin/scripts (including this script)"
    rm -rf /usr/local/bin/scripts
    REMOVED_ANYTHING=true
fi

if $REMOVED_ANYTHING; then
    log "  ✓ Old system files removed"
fi

# --- Summary ---
log ""
log "========================================="
log "Migration Complete!"
log "========================================="
log ""
log "Changes made:"
log "  • Migrated data to ${NEW_SHARE_DIR}"
log "  • Updated hyprland.conf"
log "  • Set up user crontab for ${TARGET_USER}"
log "  • Removed old system files"
log ""
log "System will reboot shortly to complete update."
log "After reboot, updates will run as ${TARGET_USER}."
log "========================================="

exit 0
