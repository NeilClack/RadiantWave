# RadiantWave v0.1.2 Deployment Guide

## How the Automatic Migration Works

When you build and deploy v0.1.2, the migration happens automatically during the user's next update. Here's exactly how it flows:

### Build-Time Setup

Your `./build.sh --release` packages two migration components:

1. **For automatic deployment** (old updater will find this):
   ```
   /usr/local/bin/scripts/00-migrate-v0.1.2.sh
   ```
   - Packaged in OLD location where old updater looks
   - Runs automatically during v0.1.2 installation
   - Self-destructs after completing migration

2. **For manual testing** (optional, for your use):
   ```
   /usr/local/bin/radiantwave-migrate-v0.1.2.sh
   ```
   - Standalone version for manual testing
   - Also cleaned up by migration script
   - Not needed for production deployment

### Deployment Flow

```
┌─────────────────────────────────────────────────────────────┐
│ User System (v0.1.0 or v0.1.1-r1)                           │
├─────────────────────────────────────────────────────────────┤
│ /usr/local/bin/radiantwave-updater (OLD, runs as root)      │
│ /usr/local/bin/radiantwave                                  │
│ /usr/local/share/radiantwave/                               │
│ /etc/crontab: "0 0 * * * root radiantwave-updater"          │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ @reboot or scheduled time
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ OLD UPDATER RUNS                                            │
├─────────────────────────────────────────────────────────────┤
│ 1. Downloads v0.1.2 tarball from repository                 │
│ 2. Extracts to / (root filesystem)                          │
│    ├─ New files → /home/localuser/.local/...                │
│    └─ Migration → /usr/local/bin/scripts/00-migrate-*.sh    │
│ 3. Looks for post-install scripts in /usr/local/bin/scripts/│
│ 4. Finds and runs: 00-migrate-v0.1.2.sh                     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ MIGRATION SCRIPT RUNS (as root)                             │
├─────────────────────────────────────────────────────────────┤
│ Step 1: Migrate data                                        │
│   ├─ Copy data.db from old → new location                   │
│   ├─ Copy VERSION file                                      │
│   ├─ Copy .sha256 checksums                                 │
│   └─ Set ownership to localuser:localuser                   │
│                                                              │
│ Step 2: Update hyprland.conf                                │
│   ├─ Backup existing config                                 │
│   └─ Install new config                                     │
│                                                              │
│ Step 3: Update crontab                                      │
│   ├─ Remove entries from /etc/crontab                       │
│   └─ Add to localuser crontab:                              │
│       @reboot /home/localuser/.local/bin/radiantwave-updater│
│                                                              │
│ Step 4: Set permissions                                     │
│   └─ chmod +x on all scripts and binaries                   │
│                                                              │
│ Step 5: Create migration marker                             │
│   └─ /home/localuser/.local/share/radiantwave/              │
│       .migration-v0.1.2-complete                            │
│                                                              │
│ Step 6: Clean up old system                                 │
│   ├─ Remove /usr/local/bin/radiantwave                      │
│   ├─ Remove /usr/local/bin/radiantwave-updater              │
│   ├─ Remove /usr/local/bin/radiantwave-migrate-v0.1.2.sh    │
│   ├─ Remove /usr/local/share/radiantwave/ (including        │
│   │   post_install scripts)                                 │
│   └─ Remove /usr/local/bin/scripts/ (including itself!)     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ OLD UPDATER COMPLETES                                       │
├─────────────────────────────────────────────────────────────┤
│ 1. Reboots system: systemctl reboot                         │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ System reboots
                            ▼
┌─────────────────────────────────────────────────────────────┐
│ AFTER REBOOT - New System (v0.1.2)                          │
├─────────────────────────────────────────────────────────────┤
│ /home/localuser/.local/bin/radiantwave-updater (NEW)        │
│ /home/localuser/.local/bin/radiantwave                      │
│ /home/localuser/.local/share/radiantwave/                   │
│ localuser crontab: "@reboot radiantwave-updater"            │
│                                                              │
│ Future updates:                                             │
│ - Run as localuser (no root)                                │
│ - Files in ~/.local/ (XDG compliant)                        │
│ - User crontab @reboot (no /etc/crontab)                    │
└─────────────────────────────────────────────────────────────┘
```

## What Gets Cleaned Up

The migration script removes all old system files:

### Binaries
- `/usr/local/bin/radiantwave`
- `/usr/local/bin/radiantwave-updater`
- `/usr/local/bin/radiantwave-migrate-v0.1.2.sh` (standalone testing version)

### Scripts
- `/usr/local/bin/scripts/` (entire directory, including migration script itself)

### Data & Assets
- `/usr/local/share/radiantwave/` (entire directory including):
  - `post_install/05-migrate-to-user-local.sh`
  - `post_install/10-update-hyprland-config.sh`
  - `post_install/90-cleanup-old-system.sh`
  - All other old system files

### Crontab
- Removes all `radiantwave-updater` entries from `/etc/crontab`
- Adds `@reboot` entry to localuser's crontab

## Testing the Deployment

### 1. Build v0.1.2

```bash
cd /home/nclack/Work/RadiantWave
./build.sh --release home release
```

This creates:
- `radiantwave-home-v0.1.2.tar.xz`
- `radiantwave-home-v0.1.2.tar.xz.sha256`
- Uploads to repository

### 2. On Test System (Running v0.1.0/v0.1.1-r1)

Wait for scheduled update OR manually trigger:

```bash
# As root (old updater runs as root)
sudo /usr/local/bin/radiantwave-updater
```

Watch the logs:
```bash
# In another terminal
journalctl -f -t radiantwave-updater -t 00-migrate-v0.1.2.sh
```

### 3. Verify Migration

After the automatic reboot:

```bash
# Check new files exist
ls -la /home/localuser/.local/bin/radiantwave*
ls -la /home/localuser/.local/share/radiantwave/

# Check old files are gone
ls /usr/local/bin/radiantwave* 2>/dev/null && echo "ERROR: Old files still present" || echo "✓ Old files removed"
ls /usr/local/share/radiantwave 2>/dev/null && echo "ERROR: Old share dir still present" || echo "✓ Old share dir removed"

# Check crontab
sudo crontab -u localuser -l | grep radiantwave  # Should show @reboot entry
sudo cat /etc/crontab | grep radiantwave  # Should be empty

# Check migration marker
cat /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete

# Test new updater (as localuser, no sudo)
su - localuser -c '/home/localuser/.local/bin/radiantwave-updater'
```

## Edge Cases Handled

### 1. Migration Already Completed
- Marker file prevents duplicate runs
- Script exits early if marker exists
- Safe to re-run if needed (delete marker first)

### 2. Partial v0.1.2 Installation
- If v0.1.2 tarball was partially extracted
- Script detects existing new files
- Completes migration without errors

### 3. Data Conflicts
- If database exists in both old and new locations
- Keeps the newer file (based on modification time)
- Preserves user data

### 4. Missing Files
- Script checks for required files before proceeding
- Logs warnings for missing optional files
- Continues with migration

## Rollback (Development/Testing Only)

If you need to test the migration again:

```bash
# Remove migration marker
sudo rm /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete

# Restore old system (if you have backup)
# ... restore old files ...

# Re-run migration
sudo /usr/local/bin/scripts/00-migrate-v0.1.2.sh
```

## Production Deployment Checklist

- [ ] Build v0.1.2 with `./build.sh --release home release`
- [ ] Verify tarball contains `/usr/local/bin/scripts/00-migrate-v0.1.2.sh`
- [ ] Upload to repository (done automatically by build.sh)
- [ ] Test on development system first
- [ ] Verify migration completes successfully
- [ ] Check logs for errors
- [ ] Verify all old files are removed
- [ ] Verify new updater runs as localuser
- [ ] Deploy to production systems
- [ ] Monitor first few systems for issues

## Monitoring Deployment

Watch for issues across your fleet:

```bash
# Check which systems have migrated
ssh user@system "test -f /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete && echo 'Migrated' || echo 'Not migrated'"

# Check for stuck systems
ssh user@system "ls -la /usr/local/bin/radiantwave* /home/localuser/.local/bin/radiantwave* 2>/dev/null"

# Review migration logs
ssh user@system "journalctl -u cron -t 00-migrate-v0.1.2.sh"
```

## Support & Troubleshooting

If migration fails on a system:

1. Check logs: `journalctl -t 00-migrate-v0.1.2.sh`
2. Verify v0.1.2 files were extracted: `ls /home/localuser/.local/bin/`
3. Check for permission issues: `ls -la /home/localuser/.local/`
4. Review marker file: `cat /home/localuser/.local/share/radiantwave/.migration-v0.1.2-complete`
5. If needed, run manual migration script: `/home/nclack/Work/RadiantWave/radiantwave-migrate-v0.1.2.sh`

See `MIGRATION-v0.1.2.md` for detailed troubleshooting steps.
