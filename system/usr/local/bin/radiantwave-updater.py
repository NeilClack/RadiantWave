#!/usr/bin/env python3
"""
radiantwave-updater.py

Auto-updater for RadiantWave kiosk application.
Uses apt to check for and install updates from the configured repository.

USAGE:
  Run as kiosk user:
  python3 /usr/local/bin/radiantwave-updater.py
"""

import os
import sys
import subprocess
from pathlib import Path
from datetime import datetime

# Configuration - templated by build script
PACKAGE_NAME = "__CHANNEL__"

# Paths
LOG_FILE = Path("/home/kiosk/radiantwave-updater.log")


class UpdaterLogger:
    """Simple logger that writes to both stdout and file"""

    def __init__(self, log_file):
        self.log_file = log_file
        log_file.parent.mkdir(parents=True, exist_ok=True)
        log_file.touch(exist_ok=True)

    def log(self, message):
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        formatted = f"[{timestamp}] {message}"
        print(formatted)
        try:
            with open(self.log_file, 'a') as f:
                f.write(formatted + '\n')
        except IOError:
            pass

    def error(self, message):
        self.log(f"ERROR: {message}")


logger = UpdaterLogger(LOG_FILE)


def run_command(cmd, description, use_pkexec=False):
    """Run a command and return success status"""
    if use_pkexec:
        cmd = ["pkexec"] + cmd
    
    logger.log(f"{description}...")
    
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300
        )

        if result.stdout:
            for line in result.stdout.splitlines():
                logger.log(f"  {line}")

        if result.returncode != 0:
            if result.stderr:
                for line in result.stderr.splitlines():
                    logger.error(f"  {line}")
            return False

        return True

    except subprocess.TimeoutExpired:
        logger.error(f"{description} timed out")
        return False
    except Exception as e:
        logger.error(f"{description} failed: {e}")
        return False


def check_for_upgrade():
    """Check if an upgrade is available for the package"""
    logger.log(f"Checking for upgrades to {PACKAGE_NAME}...")

    try:
        result = subprocess.run(
            ["apt", "list", "--upgradable"],
            capture_output=True,
            text=True,
            timeout=30
        )

        if PACKAGE_NAME in result.stdout:
            logger.log(f"Upgrade available for {PACKAGE_NAME}")
            return True
        else:
            logger.log(f"{PACKAGE_NAME} is up to date")
            return False

    except Exception as e:
        logger.error(f"Could not check for upgrades: {e}")
        return False


def main():
    logger.log("=========================================")
    logger.log("RadiantWave Updater")
    logger.log(f"Package: {PACKAGE_NAME}")
    logger.log("=========================================")

    # Update package lists
    if not run_command(["apt", "update"], "Updating package lists", use_pkexec=True):
        logger.error("Failed to update package lists")
        return 1

    # Check for upgrades
    if not check_for_upgrade():
        logger.log("No upgrade needed")
        return 0

    # Install upgrade
    if not run_command(
        ["apt", "install", "--only-upgrade", "-y", PACKAGE_NAME],
        f"Installing upgrade for {PACKAGE_NAME}",
        use_pkexec=True
    ):
        logger.error("Upgrade installation failed")
        return 1

    logger.log("Upgrade installed successfully!")

    # Restart getty to reload the application
    if not run_command(
        ["systemctl", "restart", "getty@tty1.service"],
        "Restarting application",
        use_pkexec=True
    ):
        logger.error("Failed to restart application (manual reboot may be required)")
        return 1

    logger.log("Application restarted")
    logger.log("=========================================")
    return 0


if __name__ == "__main__":
    sys.exit(main())