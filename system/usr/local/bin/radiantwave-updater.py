#!/usr/bin/env python3
"""
radiantwave-updater.py

Auto-updater for RadiantWave kiosk application.
Uses apt to check for and install updates from the configured repository.

SYSTEM PATHS:
  - Updater: /usr/local/bin/radiantwave-updater.py
  - Binary: /usr/local/bin/radiantwave
  - Assets: /usr/local/share/radiantwave/
  - Log: /home/kiosk/radiantwave-updater.log

USAGE:
  Run as kiosk user:
  python3 /usr/local/bin/radiantwave-updater.py
"""

import os
import sys
import subprocess
from pathlib import Path
from datetime import datetime

# Configuration
CURRENT_USER = "kiosk"
PACKAGE_NAME = "__CHANNEL__"  # Will be templated to "radiantwave" or "radiantwave-dev"

# Paths
SHARE_DIR = Path("/usr/local/share/radiantwave")
VERSION_FILE = SHARE_DIR / "VERSION"
LOG_FILE = Path(f"/home/{CURRENT_USER}/radiantwave-updater.log")


class UpdaterLogger:
    """Simple logger that writes to both stdout and file"""

    def __init__(self, log_file):
        self.log_file = log_file
        # Ensure log file exists and is writable
        log_file.parent.mkdir(parents=True, exist_ok=True)
        log_file.touch(exist_ok=True)

    def log(self, message):
        """Log message with timestamp"""
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        formatted = f"[{timestamp}] {message}"
        print(formatted)

        try:
            with open(self.log_file, 'a') as f:
                f.write(formatted + '\n')
        except IOError as e:
            print(f"Warning: Could not write to log file: {e}")

    def error(self, message):
        """Log error message"""
        self.log(f"ERROR: {message}")

    def die(self, message):
        """Log error and exit"""
        self.error(message)
        sys.exit(1)


# Initialize logger
logger = UpdaterLogger(LOG_FILE)


def get_current_version():
    """Read current installed version"""
    if not VERSION_FILE.exists():
        logger.log("No VERSION file found, assuming fresh install")
        return "unknown"

    try:
        with open(VERSION_FILE, 'r') as f:
            version = f.read().strip()
            logger.log(f"Current version: {version}")
            return version
    except IOError as e:
        logger.error(f"Could not read VERSION file: {e}")
        return "unknown"


def run_command(cmd, description):
    """Run a command with pkexec and return success status"""
    logger.log(f"{description}...")
    logger.log(f"Running: {' '.join(cmd)}")

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300  # 5 minute timeout
        )

        # Log output
        if result.stdout:
            for line in result.stdout.splitlines():
                logger.log(f"  {line}")

        if result.stderr and result.returncode != 0:
            for line in result.stderr.splitlines():
                logger.error(f"  {line}")

        if result.returncode != 0:
            logger.error(f"{description} failed with exit code {result.returncode}")
            return False

        logger.log(f"✓ {description} completed")
        return True

    except subprocess.TimeoutExpired:
        logger.error(f"{description} timed out")
        return False
    except Exception as e:
        logger.error(f"{description} failed: {e}")
        return False


def update_package_lists():
    """Run apt update to refresh package lists"""
    cmd = ["pkexec", "apt", "update"]
    return run_command(cmd, "Updating package lists")


def check_for_upgrade():
    """Check if an upgrade is available for the package"""
    logger.log(f"Checking for upgrades to {PACKAGE_NAME}...")

    try:
        # Use apt list to check if upgrade available
        result = subprocess.run(
            ["apt", "list", "--upgradable", PACKAGE_NAME],
            capture_output=True,
            text=True,
            timeout=30
        )

        # If package appears in upgradable list, upgrade is available
        if PACKAGE_NAME in result.stdout:
            logger.log(f"✓ Upgrade available for {PACKAGE_NAME}")
            return True
        else:
            logger.log(f"✓ {PACKAGE_NAME} is up to date")
            return False

    except Exception as e:
        logger.error(f"Could not check for upgrades: {e}")
        return False


def install_upgrade():
    """Install available upgrade using apt"""
    cmd = ["pkexec", "apt", "install", "--only-upgrade", "-y", PACKAGE_NAME]
    return run_command(cmd, f"Installing upgrade for {PACKAGE_NAME}")


def main():
    """Main updater workflow"""
    logger.log("=========================================")
    logger.log("RadiantWave Updater")
    logger.log("=========================================")
    logger.log(f"Package: {PACKAGE_NAME}")
    logger.log(f"Running as user: {os.getenv('USER', 'unknown')}")
    logger.log(f"Log file: {LOG_FILE}")
    logger.log("")

    # Get current version for informational purposes
    current_version = get_current_version()

    # Update package lists
    if not update_package_lists():
        logger.error("Failed to update package lists")
        logger.log("=========================================")
        return 1

    # Check for upgrades
    upgrade_available = check_for_upgrade()

    if not upgrade_available:
        logger.log("No upgrade needed")
        logger.log("=========================================")
        return 0

    # Install upgrade
    if install_upgrade():
        logger.log("")
        logger.log("✓ Upgrade installed successfully!")

        # Read new version
        new_version = get_current_version()
        if new_version != current_version:
            logger.log(f"  Version: {current_version} → {new_version}")

        logger.log("=========================================")
        return 0
    else:
        logger.error("Upgrade installation failed")
        logger.log("=========================================")
        return 1


if __name__ == "__main__":
    sys.exit(main())
