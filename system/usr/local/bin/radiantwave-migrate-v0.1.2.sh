#!/bin/bash
# radiantwave-migrate-v0.1.2.sh
#
# Manual migration script for RadiantWave v0.1.0/v0.1.1-r1 -> v0.1.2
# This script handles the transition from the old system-wide installation
# to the new user-local installation.
#
# OLD SYSTEM (v0.1.0, v0.1.1-r1):
#   - Binary: /usr/local/bin/radiantwave
#   - Updater: /usr/local/bin/radiantwave-updater
#   - Scripts: /usr/local/bin/scripts/
#   - Assets: /usr/local/share/radiantwave/
#   - Runs as root via /etc/crontab
#
# NEW SYSTEM (v0.1.2+):
#   - Binary: /home/localuser/.local/bin/radiantwave
#   - Updater: /home/localuser/.local/bin/radiantwave-updater
#   - Scripts: /home/localuser/.local/bin/scripts/
#   - Assets: /home/localuser/.local/share/radiantwave/
#   - Runs as localuser via user crontab
#
# USAGE:
#   Run as root: sudo bash radiantwave-migrate-v0.1.2.sh

set -euo pipefail

SCRIPT="$(basename "$0")"
log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; logger -t "$SCRIPT" "$*"; }
die() { log "ERROR: $*"; exit 1; }

# --- Configuration ---
TARGET_USER="localuser"
MIGRATION_MARKER="/home/${TARGET_USER}/.local/share/radiantwave/.migration-v0.1.2-complete"

# Old system paths
OLD_BIN_DIR="/usr/local/bin"
OLD_SHARE_DIR="/usr/local/share/radiantwave"
OLD_UPDATER="${OLD_BIN_DIR}/radiantwave-updater"
OLD_BINARY="${OLD_BIN_DIR}/radiantwave"
OLD_SCRIPTS_DIR="${OLD_BIN_DIR}/scripts"

# New system paths
NEW_HOME="/home/${TARGET_USER}"
NEW_LOCAL_DIR="${NEW_HOME}/.local"
NEW_BIN_DIR="${NEW_LOCAL_DIR}/bin"
NEW_SHARE_DIR="${NEW_LOCAL_DIR}/share/radiantwave"
NEW_UPDATER="${NEW_BIN_DIR}/radiantwave-updater"
NEW_BINARY="${NEW_BIN_DIR}/radiantwave"
NEW_SCRIPTS_DIR="${NEW_BIN_DIR}/scripts"

# Hyprland config
HYPRLAND_CONFIG_DIR="${NEW_HOME}/.config/hypr"
HYPRLAND_CONFIG="${HYPRLAND_CONFIG_DIR}/hyprland.conf"
NEW_HYPRLAND_SOURCE="${NEW_SHARE_DIR}/hyprland.conf"

# Crontab
SYSTEM_CRONTAB="/etc/crontab"

# --- Preflight checks ---
log "========================================="
log "RadiantWave v0.1.2 Migration Script"
log "========================================="

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   die "This script must be run as root (use sudo)"
fi

# Check if target user exists
if ! id "${TARGET_USER}" &>/dev/null; then
    die "User '${TARGET_USER}' does not exist"
fi

# Check if already migrated
if [[ -f "$MIGRATION_MARKER" ]]; then
    log "Migration marker found at ${MIGRATION_MARKER}"
    log "Migration appears to have already completed."
    log "If you need to re-run migration, delete the marker file first:"
    log "  sudo rm ${MIGRATION_MARKER}"
    exit 0
fi

# --- Detect current state ---
log ""
log "Detecting current system state..."

OLD_SYSTEM_PRESENT=false
NEW_SYSTEM_PRESENT=false
BROKEN_V012_PRESENT=false

if [[ -f "$OLD_UPDATER" ]] || [[ -d "$OLD_SHARE_DIR" ]]; then
    OLD_SYSTEM_PRESENT=true
    log "  ✓ Old system detected (files in /usr/local/)"
fi

if [[ -f "$NEW_UPDATER" ]] || [[ -d "$NEW_SHARE_DIR" ]]; then
    NEW_SYSTEM_PRESENT=true
    log "  ✓ New system detected (files in ${NEW_LOCAL_DIR})"
fi

# Check for broken v0.1.2 state (new updater exists but old system not cleaned up)
if $NEW_SYSTEM_PRESENT && $OLD_SYSTEM_PRESENT; then
    BROKEN_V012_PRESENT=true
    log "  ⚠ Detected partially migrated state (v0.1.2 updater may have run with bugs)"
fi

if ! $OLD_SYSTEM_PRESENT && ! $NEW_SYSTEM_PRESENT; then
    die "Neither old nor new system detected. Nothing to migrate."
fi

# --- Step 1: Install new system files if needed ---
log ""
log "Step 1: Installing new system files..."

# Check if v0.1.2 files are already extracted
if $NEW_SYSTEM_PRESENT; then
    log "  New system files already present at ${NEW_LOCAL_DIR}"
    log "  Skipping extraction (files already in place)"
else
    # Look for v0.1.2 tarball in TEMP directory
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    TEMP_DIR="${SCRIPT_DIR}/TEMP"
    V012_TARBALL="${TEMP_DIR}/radiantwave-home-v0.1.2.tar.xz"

    if [[ -f "$V012_TARBALL" ]]; then
        log "  Found v0.1.2 tarball at ${V012_TARBALL}"
        log "  Extracting to root filesystem..."
        tar --no-same-owner -xJf "$V012_TARBALL" -C / || die "Failed to extract tarball"
        log "  ✓ Extraction complete"
    else
        die "v0.1.2 tarball not found at ${V012_TARBALL}. Please ensure it's available."
    fi
fi

# Ensure new directories exist with proper ownership
log "  Setting up directory structure..."
mkdir -p "$NEW_BIN_DIR" "$NEW_SCRIPTS_DIR" "$NEW_SHARE_DIR" "$HYPRLAND_CONFIG_DIR"
chown -R "${TARGET_USER}:${TARGET_USER}" "$NEW_LOCAL_DIR"
log "  ✓ Directory structure ready"

# --- Step 2: Migrate data from old to new locations ---
log ""
log "Step 2: Migrating data from old system..."

if [[ -d "$OLD_SHARE_DIR" ]]; then
    log "  Found old data directory at ${OLD_SHARE_DIR}"

    # Special handling for database - preserve user data
    OLD_DB="${OLD_SHARE_DIR}/data.db"
    NEW_DB="${NEW_SHARE_DIR}/data.db"

    if [[ -f "$OLD_DB" ]]; then
        if [[ -f "$NEW_DB" ]]; then
            log "  ⚠ Database exists in both locations"
            log "    Old: ${OLD_DB}"
            log "    New: ${NEW_DB}"
            log "    Keeping newer database..."

            if [[ "$OLD_DB" -nt "$NEW_DB" ]]; then
                log "    Old database is newer, copying to new location"
                cp -a "$OLD_DB" "$NEW_DB"
            else
                log "    New database is newer, keeping it"
            fi
        else
            log "  Migrating database from ${OLD_DB}"
            cp -a "$OLD_DB" "$NEW_DB"
        fi
    fi

    # Migrate VERSION file
    if [[ -f "${OLD_SHARE_DIR}/VERSION" ]]; then
        log "  Migrating VERSION file"
        cp -a "${OLD_SHARE_DIR}/VERSION" "${NEW_SHARE_DIR}/VERSION"
    fi

    # Migrate .sha256 checksum files
    for shafile in "${OLD_SHARE_DIR}"/*.sha256; do
        if [[ -f "$shafile" ]]; then
            filename="$(basename "$shafile")"
            log "  Migrating checksum file: ${filename}"
            cp -a "$shafile" "${NEW_SHARE_DIR}/${filename}"
        fi
    done

    # Fix ownership of migrated files
    chown -R "${TARGET_USER}:${TARGET_USER}" "$NEW_SHARE_DIR"
    log "  ✓ Data migration complete"
else
    log "  No old data directory found, skipping data migration"
fi

# --- Step 3: Update hyprland.conf ---
log ""
log "Step 3: Updating hyprland configuration..."

if [[ -f "$NEW_HYPRLAND_SOURCE" ]]; then
    log "  Copying hyprland.conf from ${NEW_HYPRLAND_SOURCE}"
    log "  to ${HYPRLAND_CONFIG}"

    # Backup existing config if present
    if [[ -f "$HYPRLAND_CONFIG" ]]; then
        BACKUP="${HYPRLAND_CONFIG}.backup-$(date +%Y%m%d-%H%M%S)"
        log "  Backing up existing config to ${BACKUP}"
        cp "$HYPRLAND_CONFIG" "$BACKUP"
    fi

    cp "$NEW_HYPRLAND_SOURCE" "$HYPRLAND_CONFIG"
    chown "${TARGET_USER}:${TARGET_USER}" "$HYPRLAND_CONFIG"
    chmod 644 "$HYPRLAND_CONFIG"
    log "  ✓ Hyprland config updated"
else
    log "  ⚠ Warning: hyprland.conf not found at ${NEW_HYPRLAND_SOURCE}"
fi

# --- Step 4: Clean up old system files ---
log ""
log "Step 4: Cleaning up old system files..."

REMOVED_ANYTHING=false

if [[ -f "$OLD_UPDATER" ]]; then
    log "  Removing ${OLD_UPDATER}"
    rm -f "$OLD_UPDATER"
    REMOVED_ANYTHING=true
fi

if [[ -f "$OLD_BINARY" ]]; then
    log "  Removing ${OLD_BINARY}"
    rm -f "$OLD_BINARY"
    REMOVED_ANYTHING=true
fi

if [[ -d "$OLD_SCRIPTS_DIR" ]]; then
    log "  Removing ${OLD_SCRIPTS_DIR}"
    rm -rf "$OLD_SCRIPTS_DIR"
    REMOVED_ANYTHING=true
fi

# Also check for intermediate location that may have existed
if [[ -d "${OLD_BIN_DIR}/radiantwave" ]]; then
    log "  Removing ${OLD_BIN_DIR}/radiantwave directory"
    rm -rf "${OLD_BIN_DIR}/radiantwave"
    REMOVED_ANYTHING=true
fi

if [[ -d "$OLD_SHARE_DIR" ]]; then
    log "  Removing ${OLD_SHARE_DIR}"
    rm -rf "$OLD_SHARE_DIR"
    REMOVED_ANYTHING=true
fi

if $REMOVED_ANYTHING; then
    log "  ✓ Old system files removed"
else
    log "  No old system files found to remove"
fi

# --- Step 5: Update crontab configuration ---
log ""
log "Step 5: Updating crontab configuration..."

# Remove old system crontab entry
if [[ -f "$SYSTEM_CRONTAB" ]]; then
    if grep -q "radiantwave-updater" "$SYSTEM_CRONTAB"; then
        log "  Removing old updater entries from ${SYSTEM_CRONTAB}"

        # Backup system crontab
        cp "$SYSTEM_CRONTAB" "${SYSTEM_CRONTAB}.backup-$(date +%Y%m%d-%H%M%S)"

        # Remove all lines containing radiantwave-updater
        sed -i '/radiantwave-updater/d' "$SYSTEM_CRONTAB"
        log "  ✓ Removed old crontab entries"
    else
        log "  No old radiantwave-updater entries found in system crontab"
    fi
else
    log "  System crontab not found"
fi

# Add new user crontab entry for localuser
log "  Setting up user crontab for ${TARGET_USER}..."

# Get existing crontab for localuser (may fail if no crontab exists yet)
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

log "  ✓ User crontab configured with @reboot updater"
log "    Entry: @reboot ${NEW_UPDATER}"

# Display current user crontab for verification
log ""
log "  Current crontab for ${TARGET_USER}:"
crontab -u "${TARGET_USER}" -l | sed 's/^/    /'

# --- Step 6: Set executable permissions ---
log ""
log "Step 6: Setting executable permissions..."

if [[ -f "$NEW_BINARY" ]]; then
    chmod +x "$NEW_BINARY"
    log "  ✓ Set executable: ${NEW_BINARY}"
fi

if [[ -f "$NEW_UPDATER" ]]; then
    chmod +x "$NEW_UPDATER"
    log "  ✓ Set executable: ${NEW_UPDATER}"
fi

if [[ -d "$NEW_SCRIPTS_DIR" ]]; then
    for script in "${NEW_SCRIPTS_DIR}"/*.sh; do
        if [[ -f "$script" ]]; then
            chmod +x "$script"
            log "  ✓ Set executable: $(basename "$script")"
        fi
    done
fi

# --- Step 7: Create migration marker ---
log ""
log "Step 7: Creating migration marker..."

mkdir -p "$(dirname "$MIGRATION_MARKER")"
{
    echo "Migration completed: $(date '+%Y-%m-%d %H:%M:%S')"
    echo "Migrated from: Old system (/usr/local/)"
    echo "Migrated to: New system (${NEW_LOCAL_DIR})"
    echo "Script: $SCRIPT"
} > "$MIGRATION_MARKER"
chown "${TARGET_USER}:${TARGET_USER}" "$MIGRATION_MARKER"

log "  ✓ Migration marker created at ${MIGRATION_MARKER}"

# --- Summary ---
log ""
log "========================================="
log "Migration Complete!"
log "========================================="
log ""
log "Summary of changes:"
log "  • Installed v0.1.2 files to ${NEW_LOCAL_DIR}"
log "  • Migrated user data from ${OLD_SHARE_DIR} to ${NEW_SHARE_DIR}"
log "  • Updated hyprland.conf at ${HYPRLAND_CONFIG}"
log "  • Removed old system files from /usr/local/"
log "  • Removed old crontab entries from ${SYSTEM_CRONTAB}"
log "  • Added @reboot updater to ${TARGET_USER}'s crontab"
log "  • Set proper permissions and ownership"
log ""
log "Next steps:"
log "  1. Review the changes above"
log "  2. Test the new updater manually (as ${TARGET_USER}):"
log "     su - ${TARGET_USER} -c '${NEW_UPDATER}'"
log "  3. Test the application:"
log "     su - ${TARGET_USER} -c '${NEW_BINARY}'"
log "  4. If everything works, reboot to test @reboot cron entry"
log "  5. After confirming everything works, you can merge this script"
log "     into the regular updater flow"
log ""
log "To undo this migration (for testing):"
log "  sudo rm ${MIGRATION_MARKER}"
log ""
log "========================================="

exit 0
