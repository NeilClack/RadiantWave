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

    def debug(self, message):
        self.log(f"DEBUG: {message}")

    def success(self, message):
        self.log(f"✓ {message}")


logger = UpdaterLogger(LOG_FILE)


def check_network(timeout=2):
    """Quick check for internet connectivity."""
    logger.log("Checking network connectivity...")
    try:
        socket.create_connection(("1.1.1.1", 53), timeout=timeout)
        logger.success("Network check passed (1.1.1.1:53 reachable)")
        return True
    except OSError as e:
        logger.error(f"Network check failed: {e}")
        return False


def run_command(cmd, description, use_pkexec=False):
    """Run a command and return success status"""
    if use_pkexec:
        cmd = ["pkexec"] + cmd
    
    logger.log(f"{description}...")
    logger.debug(f"Command: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300
        )

        logger.debug(f"Return code: {result.returncode}")

        if result.stdout:
            for line in result.stdout.splitlines():
                logger.log(f"  {line}")

        if result.returncode != 0:
            logger.error(f"{description} failed with return code {result.returncode}")
            if result.stderr:
                for line in result.stderr.splitlines():
                    logger.error(f"  {line}")
            return False

        logger.success(f"{description} completed successfully")
        return True

    except subprocess.TimeoutExpired:
        logger.error(f"{description} timed out")
        return False
    except Exception as e:
        logger.error(f"{description} failed: {e}")
        return False


def is_tailscale_installed():
    """Check if Tailscale is installed."""
    logger.log("Checking if Tailscale is installed...")
    try:
        result = subprocess.run(
            ["tailscale", "version"],
            capture_output=True,
            text=True,
            check=True
        )
        version = result.stdout.strip()
        logger.success(f"Tailscale is installed: {version}")
        return True
    except FileNotFoundError:
        logger.log("Tailscale binary not found")
        return False
    except subprocess.CalledProcessError as e:
        logger.error(f"Tailscale version check failed: {e}")
        return False


def install_tailscale():
    """Install Tailscale."""
    logger.log("Installing Tailscale...")
    try:
        logger.debug("Running: pkexec sh -c 'curl -fsSL https://tailscale.com/install.sh | sh'")
        result = subprocess.run(
            ["pkexec", "sh", "-c", "curl -fsSL https://tailscale.com/install.sh | sh"],
            capture_output=True,
            text=True,
            timeout=120
        )
        logger.debug(f"Install return code: {result.returncode}")
        if result.stdout:
            for line in result.stdout.splitlines():
                logger.log(f"  {line}")
        if result.stderr:
            for line in result.stderr.splitlines():
                logger.log(f"  [stderr] {line}")
        
        if result.returncode != 0:
            logger.error("Tailscale installation failed")
            return False
        
        logger.success("Tailscale installed successfully")
        return True
    except subprocess.TimeoutExpired:
        logger.error("Tailscale installation timed out")
        return False
    except subprocess.CalledProcessError as e:
        logger.error(f"Tailscale installation failed: {e}")
        if e.stderr:
            logger.error(f"stderr: {e.stderr}")
        return False


def set_tailscale_operator():
    """Set Tailscale operator to kiosk."""
    logger.log("Setting Tailscale operator to 'kiosk'...")
    try:
        result = subprocess.run(
            ["pkexec", "tailscale", "set", "--operator=kiosk"],
            capture_output=True,
            text=True,
            timeout=30
        )
        logger.debug(f"Return code: {result.returncode}")
        if result.stdout:
            logger.debug(f"stdout: {result.stdout}")
        if result.stderr:
            logger.debug(f"stderr: {result.stderr}")
        
        if result.returncode != 0:
            logger.error(f"Failed to set operator: {result.stderr}")
            return False
        
        logger.success("Tailscale operator set to 'kiosk'")
        return True
    except Exception as e:
        logger.error(f"Failed to set Tailscale operator: {e}")
        return False


def get_tailscale_status():
    """Get full Tailscale status as dict."""
    logger.log("Getting Tailscale status...")
    try:
        result = subprocess.run(
            ["tailscale", "status", "--json"],
            capture_output=True,
            text=True,
            timeout=10
        )
        logger.debug(f"Return code: {result.returncode}")
        
        if result.returncode != 0:
            logger.error(f"tailscale status failed: {result.stderr}")
            return None
        
        status = json.loads(result.stdout)
        
        # Log key status info
        backend_state = status.get("BackendState", "unknown")
        logger.log(f"  BackendState: {backend_state}")
        
        self_info = status.get("Self", {})
        if self_info:
            logger.log(f"  HostName: {self_info.get('HostName', 'unknown')}")
            logger.log(f"  DNSName: {self_info.get('DNSName', 'unknown')}")
            logger.log(f"  Online: {self_info.get('Online', 'unknown')}")
            tailscale_ips = self_info.get("TailscaleIPs", [])
            logger.log(f"  TailscaleIPs: {tailscale_ips}")
        
        # Check for health issues
        health = status.get("Health", [])
        if health:
            logger.log("  Health issues:")
            for issue in health:
                logger.log(f"    - {issue}")
        
        return status
    except json.JSONDecodeError as e:
        logger.error(f"Failed to parse Tailscale status JSON: {e}")
        return None
    except subprocess.TimeoutExpired:
        logger.error("Tailscale status timed out")
        return None
    except Exception as e:
        logger.error(f"Failed to get Tailscale status: {e}")
        return None


def is_tailscale_logged_in(status=None):
    """Check if Tailscale is logged in."""
    if status is None:
        status = get_tailscale_status()
    
    if status is None:
        logger.error("Cannot determine login status - no status available")
        return False
    
    backend_state = status.get("BackendState", "")
    logged_in = backend_state == "Running"
    
    if logged_in:
        logger.success("Tailscale is logged in (BackendState=Running)")
    else:
        logger.log(f"Tailscale is NOT logged in (BackendState={backend_state})")
    
    return logged_in


def tailscale_login(hostname=None):
    """Log in to Tailscale/Headscale."""
    logger.log("Logging in to Headscale...")
    logger.debug(f"  Server: {HEADSCALE_URL}")
    logger.debug(f"  AuthKey: {HEADSCALE_AUTHKEY[:8]}...{HEADSCALE_AUTHKEY[-4:]}")
    if hostname:
        logger.debug(f"  Hostname: {hostname}")
    
    cmd = [
        "tailscale", "up",
        "--login-server", HEADSCALE_URL,
        "--authkey", HEADSCALE_AUTHKEY
    ]
    if hostname:
        cmd.extend(["--hostname", hostname])
    
    logger.debug(f"Command: {' '.join(cmd)}")
    
    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=30
        )
        logger.debug(f"Return code: {result.returncode}")
        if result.stdout:
            logger.log(f"  stdout: {result.stdout}")
        if result.stderr:
            logger.log(f"  stderr: {result.stderr}")
        
        if result.returncode != 0:
            logger.error(f"Tailscale login failed with code {result.returncode}")
            return False
        
        logger.success("Tailscale login successful")
        return True
    except subprocess.TimeoutExpired:
        logger.error("Tailscale login timed out")
        return False
    except Exception as e:
        logger.error(f"Tailscale login failed: {e}")
        return False


def get_current_tailscale_hostname(status=None):
    """Get the hostname we're currently advertising."""
    if status is None:
        status = get_tailscale_status()
    
    if status is None:
        return None
    
    hostname = status.get("Self", {}).get("HostName")
    logger.debug(f"Current Tailscale hostname: {hostname}")
    return hostname


def get_license_key():
    """Fetch license key from database."""
    logger.log(f"Fetching license key from database...")
    logger.debug(f"  DB path: {DB_PATH}")
    
    if not DB_PATH.exists():
        logger.log(f"  Database does not exist")
        return None
    
    try:
        conn = sqlite3.connect(DB_PATH)
        logger.debug("  Connected to database")
        
        cursor = conn.execute("SELECT value FROM configs WHERE key = 'license_key'")
        row = cursor.fetchone()
        conn.close()
        
        if row and row[0]:
            logger.success(f"License key found: {row[0]}")
            return row[0]
        else:
            logger.log("  License key not found or empty")
            return None
    except sqlite3.Error as e:
        logger.error(f"Database error: {e}")
        return None


def ensure_tailscale_installed():
    """Ensure Tailscale is installed, install if needed."""
    logger.log("=== Ensuring Tailscale is installed ===")
    
    if is_tailscale_installed():
        return True
    
    logger.log("Tailscale not installed, attempting installation...")
    
    if not install_tailscale():
        return False
    
    # Verify installation
    if not is_tailscale_installed():
        logger.error("Tailscale installation verification failed")
        return False
    
    # Set operator
    if not set_tailscale_operator():
        logger.error("Failed to set Tailscale operator, continuing anyway...")
    
    return True


def ensure_tailscale_connected():
    """Ensure Tailscale is logged in and connected."""
    logger.log("=== Ensuring Tailscale is connected ===")
    
    status = get_tailscale_status()
    
    if not is_tailscale_logged_in(status):
        logger.log("Tailscale not logged in, attempting login...")
        
        # Get license key to use as hostname if available
        license_key = get_license_key()
        
        if not tailscale_login(hostname=license_key):
            logger.error("Failed to log in to Tailscale")
            return False
        
        # Re-check status after login
        status = get_tailscale_status()
        if not is_tailscale_logged_in(status):
            logger.error("Still not logged in after login attempt")
            return False
    
    logger.success("Tailscale is connected")
    return True


def sync_tailscale_hostname():
    """Update tailscale hostname to match license key if needed."""
    logger.log("=== Syncing Tailscale hostname ===")
    
    license_key = get_license_key()
    
    if not license_key:
        logger.log("No license key found, skipping hostname sync")
        return
    
    status = get_tailscale_status()
    current = get_current_tailscale_hostname(status)
    
    logger.log(f"  Current hostname: {current}")
    logger.log(f"  Desired hostname: {license_key}")
    
    if current == license_key:
        logger.success(f"Hostname already correct: {license_key}")
        return
    
    logger.log(f"Hostname mismatch, updating: {current} -> {license_key}")
    
    if tailscale_login(hostname=license_key):
        logger.success("Hostname updated successfully")
        # Verify the change
        new_status = get_tailscale_status()
        new_hostname = get_current_tailscale_hostname(new_status)
        logger.log(f"  Verified new hostname: {new_hostname}")
    else:
        logger.error("Failed to update hostname")


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
        
        logger.debug(f"apt list output: {result.stdout}")

        if PACKAGE_NAME in result.stdout:
            logger.success(f"Upgrade available for {PACKAGE_NAME}")
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
    logger.log(f"Timestamp: {datetime.now().isoformat()}")
    logger.log("=========================================")

    # Quick network check
    if not check_network():
        logger.log("No internet connection, skipping update")
        return 0

    # Ensure Tailscale is installed
    if not ensure_tailscale_installed():
        logger.error("Tailscale installation failed, continuing with update check...")
    else:
        # Ensure Tailscale is connected
        if not ensure_tailscale_connected():
            logger.error("Tailscale connection failed, continuing with update check...")
        else:
            # Sync hostname with license key
            sync_tailscale_hostname()

    logger.log("=========================================")
    logger.log("=== Package Update Check ===")
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

    logger.success("Upgrade installed successfully!")

    # Restart getty to reload the application
    if not run_command(
        ["systemctl", "restart", "getty@tty1.service"],
        "Restarting application",
        use_pkexec=True
    ):
        logger.error("Failed to restart application (manual reboot may be required)")
        return 1

    logger.success("Application restarted")
    logger.log("=========================================")
    logger.log("=== Update Complete ===")
    logger.log("=========================================")
    return 0


if __name__ == "__main__":
    sys.exit(main())