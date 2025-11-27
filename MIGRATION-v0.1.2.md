# RadiantWave v0.1.2 Migration Guide

## Overview

Version 0.1.2 introduces a significant architectural change: moving from a system-wide installation (`/usr/local/`) to a user-local installation (`~/.local/`). This migration is necessary for:

1. **XDG Base Directory compliance** - Following Linux standards for user applications
2. **Security** - Running without root privileges
3. **User isolation** - Each user has their own data and configuration

## System Changes

### Old System (v0.1.0, v0.1.1-r1)

```
Location                                Purpose
────────────────────────────────────────────────────────────────
/usr/local/bin/radiantwave              Application binary
/usr/local/bin/radiantwave-updater      Update script (runs as root)
/usr/local/bin/scripts/                 Post-install scripts
/usr/local/share/radiantwave/           Assets, data, database
/usr/local/share/radiantwave/data.db    User database
/etc/crontab                            @reboot updater (as root)
```

### New System (v0.1.2+)

```
Location                                           Purpose
────────────────────────────────────────────────────────────────────────
/home/localuser/.local/bin/radiantwave             Application binary
/home/localuser/.local/bin/radiantwave-updater     Update script (user)
/home/localuser/.local/bin/scripts/                Post-install scripts
/home/localuser/.local/share/radiantwave/          Assets, data, database
/home/localuser/.local/share/radiantwave/data.db   User database
~localuser crontab                                 @reboot updater (user)
```

## Migration Script

The migration script `radiantwave-migrate-v0.1.2.sh` performs a complete transition from the old to new system.

### What the Script Does

1. **Preflight Checks**
   - Verifies running as root
   - Checks if `localuser` exists
   - Detects current system state (old/new/partially migrated)
   - Checks for migration marker to prevent duplicate runs

2. **Install New System Files**
   - Extracts v0.1.2 tarball from `TEMP/radiantwave-home-v0.1.2.tar.xz`
   - Creates directory structure at `/home/localuser/.local/`
   - Sets proper ownership to `localuser:localuser`

3. **Migrate Data**
   - **Database**: Copies `data.db` from old to new location
     - If database exists in both locations, keeps the newer one
     - Preserves user settings, affirmations, logs
   - **VERSION file**: Migrates version tracking
   - **Checksum files**: Migrates `.sha256` files for update verification

4. **Update Hyprland Configuration**
   - Copies new `hyprland.conf` to `/home/localuser/.config/hypr/`
   - Backs up existing config with timestamp
   - Sets proper ownership and permissions (644)

5. **Clean Up Old System**
   - Removes `/usr/local/bin/radiantwave`
   - Removes `/usr/local/bin/radiantwave-updater`
   - Removes `/usr/local/bin/scripts/`
   - Removes `/usr/local/share/radiantwave/`

6. **Update Crontab**
   - **System crontab (`/etc/crontab`)**:
     - Backs up existing file
     - Removes all `radiantwave-updater` entries
   - **User crontab (localuser)**:
     - Removes any existing `radiantwave-updater` entries
     - Adds: `@reboot /home/localuser/.local/bin/radiantwave-updater`

7. **Set Permissions**
   - Makes binaries executable
   - Makes scripts executable
   - Ensures all files owned by `localuser:localuser`

8. **Create Migration Marker**
   - Creates `/home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete`
   - Prevents duplicate migrations
   - Contains timestamp and migration details

## Testing the Migration

### Prerequisites

1. Have a system running v0.1.0 or v0.1.1-r1
2. Have the v0.1.2 tarball in `TEMP/radiantwave-home-v0.1.2.tar.xz`
3. SSH or console access to the target system

### Testing Procedure

```bash
# 1. Upload the migration script to the target system
scp radiantwave-migrate-v0.1.2.sh localuser@target-system:/tmp/

# 2. SSH into the target system
ssh localuser@target-system

# 3. Run the migration script as root
sudo bash /tmp/radiantwave-migrate-v0.1.2.sh

# 4. Review the output carefully
# The script provides detailed logging of every action

# 5. Verify the new updater works
su - localuser -c '/home/localuser/.local/bin/radiantwave-updater'

# 6. Verify the application works
su - localuser -c '/home/localuser/.local/bin/radiantwave'

# 7. Check user crontab
sudo crontab -u localuser -l
# Should show: @reboot /home/localuser/.local/bin/radiantwave-updater

# 8. Check system crontab
sudo cat /etc/crontab | grep radiantwave
# Should return nothing (old entries removed)

# 9. Verify data migration
ls -la /home/localuser/.local/share/radiantwave/
# Should show data.db, VERSION, and .sha256 files

# 10. Verify old system is cleaned up
ls /usr/local/bin/radiantwave* 2>/dev/null || echo "Old files removed ✓"
ls /usr/local/share/radiantwave 2>/dev/null || echo "Old share dir removed ✓"

# 11. Test @reboot functionality
sudo reboot
# After reboot, check logs to verify updater ran at boot
journalctl -t radiantwave-updater | tail -20
```

### Expected Outcomes

**Success indicators:**
- ✓ Script completes without errors
- ✓ New files present in `/home/localuser/.local/`
- ✓ Old files removed from `/usr/local/`
- ✓ Database migrated with all user data intact
- ✓ User crontab contains `@reboot` entry
- ✓ System crontab has no radiantwave entries
- ✓ Migration marker exists
- ✓ Application runs successfully as localuser
- ✓ Updater runs successfully as localuser

**Failure indicators:**
- ✗ Script exits with error
- ✗ Files missing in new location
- ✗ Old files still present in `/usr/local/`
- ✗ Database not migrated
- ✗ Permission errors when running as localuser

### Handling Broken v0.1.2 State

If the broken v0.1.2 updater has already run on the system:

**Symptoms:**
- New files exist at `/home/localuser/.local/`
- Old files still exist at `/usr/local/`
- Database may exist in both locations
- System is in partially migrated state

**Resolution:**
The migration script handles this automatically:
1. Detects both old and new systems present
2. Logs warning about partially migrated state
3. Keeps newer database if exists in both locations
4. Completes the cleanup that v0.1.2 failed to do
5. Properly configures crontab

## Rollback Procedure

If you need to rollback for testing:

```bash
# 1. Remove migration marker
sudo rm /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete

# 2. Remove new system files (optional - only if you want full rollback)
sudo rm -rf /home/localuser/.local/bin/radiantwave*
sudo rm -rf /home/localuser/.local/share/radiantwave

# 3. Remove user crontab entry
sudo crontab -u localuser -e
# Delete the @reboot line manually

# 4. Restore from backup if needed
# If you saved backups of old system files, restore them now

# 5. Re-run migration script
sudo bash radiantwave-migrate-v0.1.2.sh
```

## Post-Migration Steps

After successful migration:

1. **Monitor first few updates**
   ```bash
   # Watch updater logs
   journalctl -u cron -f | grep radiantwave-updater
   ```

2. **Verify user can run application**
   ```bash
   # Application should start without sudo
   /home/localuser/.local/bin/radiantwave
   ```

3. **Confirm automatic updates work**
   - Wait for next scheduled update OR
   - Trigger update manually: `/home/localuser/.local/bin/radiantwave-updater`

4. **Merge into production updater**
   - Once migration is confirmed working
   - Incorporate migration logic into the main updater
   - Deploy to all systems

## Integration into Production Updater

After successful testing, the migration logic should be integrated into the main updater script:

### Option 1: One-Time Migration Script

Include `05-cleanup-old-system.sh` in the v0.1.2 package's `scripts/` directory. This runs automatically as a post-install script. **This is the current approach in v0.1.2.**

### Option 2: Enhanced Updater

Modify the old updater (still in `/usr/local/bin/radiantwave-updater`) to:
1. Detect if it's running the final update (to v0.1.2+)
2. Run full migration after extracting v0.1.2
3. Clean up old system
4. Set up new crontab
5. Exit without rebooting (let user reboot manually)

### Recommended Approach

Use **Option 1** with a corrected `05-cleanup-old-system.sh`:

The current script in v0.1.2 has the right idea but needs fixes:
- ✓ It migrates data
- ✓ It removes old files
- ✓ It cleans up system crontab
- ✗ **Missing**: Setup of user crontab with `@reboot` entry
- ✗ **Issue**: Runs after broken extraction to `/` instead of proper extraction

**Fix required:** Update `05-cleanup-old-system.sh` to add:

```bash
# Add this to the cleanup script after line 88:

# --- 5) Setup user crontab ---
log "Setting up user crontab for ${TARGET_USER}..."
TEMP_CRON=$(mktemp)
if crontab -u localuser -l &>/dev/null; then
    crontab -u localuser -l > "$TEMP_CRON"
    sed -i '/radiantwave-updater/d' "$TEMP_CRON"
fi
echo "@reboot /home/localuser/.local/bin/radiantwave-updater" >> "$TEMP_CRON"
crontab -u localuser "$TEMP_CRON"
rm -f "$TEMP_CRON"
log "User crontab configured with @reboot updater"
```

## Troubleshooting

### Issue: "User 'localuser' does not exist"

**Cause:** System doesn't have the expected user account

**Fix:**
```bash
sudo useradd -m localuser
sudo passwd localuser
```

### Issue: "Permission denied" when running as localuser

**Cause:** Files not owned by localuser

**Fix:**
```bash
sudo chown -R localuser:localuser /home/localuser/.local
sudo chmod +x /home/localuser/.local/bin/radiantwave*
```

### Issue: Database not found after migration

**Cause:** Database wasn't in old location or migration failed

**Fix:**
```bash
# Check old location
sudo ls -la /usr/local/share/radiantwave/data.db

# If exists, manually copy
sudo cp /usr/local/share/radiantwave/data.db \
       /home/localuser/.local/share/radiantwave/data.db
sudo chown localuser:localuser \
       /home/localuser/.local/share/radiantwave/data.db
```

### Issue: Crontab entry not working

**Cause:** Cron service not running or entry syntax wrong

**Fix:**
```bash
# Check cron service
sudo systemctl status cron

# Verify entry syntax (no leading spaces/tabs)
sudo crontab -u localuser -l

# Test updater manually
su - localuser -c '/home/localuser/.local/bin/radiantwave-updater'
```

### Issue: Migration runs multiple times

**Cause:** Migration marker not created or was deleted

**Fix:**
```bash
# Check for marker
ls -la /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete

# Manually create if needed
sudo bash -c 'echo "Manual marker created $(date)" > \
  /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete'
sudo chown localuser:localuser \
  /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete
```

## Technical Notes

### Why This Migration is Needed

1. **Security**: Running as root is unnecessary and risky
2. **Standards**: XDG Base Directory Specification compliance
3. **Multi-user**: Allows multiple users to run RadiantWave
4. **Simplicity**: No sudo required for normal operation

### Crontab Syntax

```bash
# Old system (in /etc/crontab):
0 0 * * * root /usr/local/bin/radiantwave-updater

# New system (in user crontab):
@reboot /home/localuser/.local/bin/radiantwave-updater
```

Key differences:
- Old: Runs daily at midnight, as root
- New: Runs at every boot, as localuser

### File Ownership

All files under `/home/localuser/.local/` **must** be owned by `localuser:localuser`. The migration script enforces this with:

```bash
chown -R localuser:localuser /home/localuser/.local
```

### Migration Marker Purpose

The marker file prevents:
- Duplicate migrations
- Data loss from repeated migrations
- Unnecessary processing on subsequent updates

It should only be deleted for:
- Testing migration script changes
- Forced re-migration after rollback
- Recovery from failed migration

## Questions?

If you encounter issues not covered here, check:
1. Script output logs (very detailed)
2. System logs: `journalctl -t radiantwave-migrate-v0.1.2.sh`
3. Application logs: `/home/localuser/.local/share/radiantwave/logs.log`
4. Updater logs: `journalctl -t radiantwave-updater`
