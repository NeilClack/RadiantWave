#!/bin/bash
# Comment out the midnight RadiantWave updater job in /etc/crontab if active

CRON_FILE="/etc/crontab"
BACKUP="${CRON_FILE}.bak"

# Backup the file just in case
cp "$CRON_FILE" "$BACKUP"

# Comment the midnight updater line if it's not already commented
sed -i -E 's|^([[:space:]]*)0 0 \* \* \* root /usr/local/bin/radiantwave-updater$|\1# 0 0 * * * root /usr/local/bin/radiantwave-updater|' "$CRON_FILE"
