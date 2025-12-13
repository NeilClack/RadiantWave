#!/usr/bin/env python3
"""
radiantwave-updater.py

Auto-updater for RadiantWave kiosk application.
Uses apt to check for and install updates from the configured repository.

USAGE:
  Run as kiosk user:
  python3 /usr/local/bin/radiantwave-updater.py
"""

import sys
import subprocess
import json
import sqlite3
import socket
from pathlib import Path
from datetime import datetime

# Configuration - templated by build script
PACKAGE_NAME = "__CHANNEL__"
HEADSCALE_URL = "https://headscale.radiantwavetech.com"
HEADSCALE_AUTHKEY = "803130d5b2e38bffa9fa5b6921a3d6403100e2e30fea1bd5"

# Paths
LOG_FILE = Path("/home/kiosk/radiantwave-updater.log")
DB_PATH = Path("/usr/local/share/radiantwave/data.db")


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


def check_network(timeout=2):
    """Quick check for internet connectivity."""
    try:
        socket.create_connection(("1.1.1.1", 53), timeout=timeout)
        return True
    except OSError:
        return False


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


def get_current_tailscale_hostname():
    """Get the hostname we're currently advertising."""
    try:
        result = subprocess.run(
            ["tailscale", "status", "--json"],
            capture_output=True, text=True, check=True,
            timeout=10
        )
        status = json.loads(result.stdout)
        return status.get("Self", {}).get("HostName")
    except (subprocess.CalledProcessError, json.JSONDecodeError, subprocess.TimeoutExpired):
        return None


def get_license_key():
    """Fetch license key from database."""
    if not DB_PATH.exists():
        return None
    
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.execute("SELECT value FROM configs WHERE key = 'license_key'")
        row = cursor.fetchone()
        conn.close()
        return row[0] if row and row[0] else None
    except sqlite3.Error:
        return None


def sync_tailscale_hostname():
    """Update tailscale hostname to match license key if needed."""
    license_key = get_license_key()
    
    if not license_key:
        logger.log("No license key found, skipping tailscale hostname sync")
        return
    
    current = get_current_tailscale_hostname()
    
    if current == license_key:
        logger.log(f"Tailscale hostname already set to {license_key}")
        return
    
    logger.log(f"Updating tailscale hostname: {current} -> {license_key}")
    
    try:
        subprocess.run([
            "tailscale", "up",
            "--login-server", HEADSCALE_URL,
            "--authkey", HEADSCALE_AUTHKEY,
            "--hostname", license_key
        ], check=True, capture_output=True, text=True, timeout=30)
        logger.log("Tailscale hostname updated successfully")
    except subprocess.CalledProcessError as e:
        logger.error(f"Failed to update tailscale hostname: {e.stderr}")
    except subprocess.TimeoutExpired:
        logger.error("Tailscale hostname update timed out")


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

    # Quick network check
    if not check_network():
        logger.log("No internet connection, skipping update")
        return 0

    # Sync tailscale hostname with license key
    sync_tailscale_hostname()

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