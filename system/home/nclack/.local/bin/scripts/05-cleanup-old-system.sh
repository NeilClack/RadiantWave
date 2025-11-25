#!/bin/bash
set -euo pipefail

SCRIPT="$(basename "$0")"
log() { echo "$*"; logger -t "$SCRIPT" "$*"; }

# --- Migration marker ---
MARKER_DIR="/home/localuser/.local/share/radiantwave"
MARKER_FILE="${MARKER_DIR}/.migration-v0.1.2-complete"

# Check if migration already completed
if [[ -f "$MARKER_FILE" ]]; then
  log "Migration already complete (marker found at ${MARKER_FILE}). Skipping cleanup."
  exit 0
fi

log "Starting cleanup of old system files from /usr/local/"

# Track if we actually removed anything
REMOVED_SOMETHING=false

# --- 1) Remove old binary and updater ---
if [[ -f /usr/local/bin/radiantwave-updater ]]; then
  log "Removing /usr/local/bin/radiantwave-updater"
  rm -f /usr/local/bin/radiantwave-updater
  REMOVED_SOMETHING=true
fi

if [[ -f /usr/local/bin/radiantwave ]]; then
  log "Removing /usr/local/bin/radiantwave"
  rm -f /usr/local/bin/radiantwave
  REMOVED_SOMETHING=true
fi

# --- 2) Remove old scripts directory ---
if [[ -d /usr/local/bin/scripts ]]; then
  log "Removing /usr/local/bin/scripts/ (including all subdirectories)"
  rm -rf /usr/local/bin/scripts
  REMOVED_SOMETHING=true
fi

# Also check for the intermediate location (radiantwave/scripts) that existed briefly
if [[ -d /usr/local/bin/radiantwave ]]; then
  log "Removing /usr/local/bin/radiantwave/ directory"
  rm -rf /usr/local/bin/radiantwave
  REMOVED_SOMETHING=true
fi

# --- 3) Migrate old data directory to new location ---
OLD_DATA_DIR="/usr/local/share/radiantwave"
NEW_DATA_DIR="/home/localuser/.local/share/radiantwave"

if [[ -d "$OLD_DATA_DIR" ]]; then
  log "Found old data directory at ${OLD_DATA_DIR}"

  # Ensure new directory exists
  mkdir -p "$NEW_DATA_DIR"

  # Migrate files from old location to new location
  # Only copy if files don't exist in new location (don't overwrite newer files)
  for file in "$OLD_DATA_DIR"/*; do
    [[ -e "$file" ]] || continue  # Skip if glob didn't match anything

    filename="$(basename "$file")"
    if [[ ! -e "${NEW_DATA_DIR}/${filename}" ]]; then
      log "Migrating ${filename} to ${NEW_DATA_DIR}/"
      cp -a "$file" "$NEW_DATA_DIR/"
    else
      log "Skipping ${filename} (already exists in new location)"
    fi
  done

  # Set proper ownership for migrated files
  chown -R localuser:localuser "$NEW_DATA_DIR"

  # Remove old data directory after successful migration
  log "Removing old data directory ${OLD_DATA_DIR}"
  rm -rf "$OLD_DATA_DIR"
  REMOVED_SOMETHING=true
fi

# --- 4) Clean up old cronjob references (if any still exist) ---
CRON_FILE="/etc/crontab"
if [[ -f "$CRON_FILE" ]] && grep -q "^[[:space:]]*0 0 \* \* \* root /usr/local/bin/radiantwave-updater" "$CRON_FILE"; then
  log "Removing old cronjob reference from ${CRON_FILE}"
  sed -i '/^[[:space:]]*0 0 \* \* \* root \/usr\/local\/bin\/radiantwave-updater/d' "$CRON_FILE"
  REMOVED_SOMETHING=true
fi

# --- 5) Create migration marker ---
mkdir -p "$MARKER_DIR"
date '+%Y-%m-%d %H:%M:%S' > "$MARKER_FILE"
chown localuser:localuser "$MARKER_FILE"

if $REMOVED_SOMETHING; then
  log "Cleanup complete. Old system files removed and migration marker created."
else
  log "No old system files found. Migration marker created."
fi

log "Migration from /usr/local/ to /home/localuser/.local/ complete."
exit 0
