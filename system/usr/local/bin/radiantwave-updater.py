#!/usr/bin/env python3
"""
radiantwave-updater.py

Auto-updater for RadiantWave kiosk application.
Checks for updates, downloads, verifies, and installs new versions.

SYSTEM PATHS:
  - Updater: /usr/local/bin/radiantwave-updater.py
  - Binary: /usr/local/bin/radiantwave
  - Scripts: /usr/local/bin/scripts/
  - Assets: /usr/local/share/radiantwave/
  - Log: /home/kiosk/radiantwave-updater.log

USAGE:
  Run as kiosk user:
  python3 /usr/local/bin/radiantwave-updater.py
"""

import os
import sys
import shutil
import subprocess
import tempfile
from pathlib import Path
from datetime import datetime
import hashlib
import urllib.request
import urllib.error

# Configuration
UPDATE_SERVER = "https://repository.radiantwavetech.com"
CURRENT_USER = "kiosk"

# System paths
BIN_DIR = Path("/usr/local/bin")
SHARE_DIR = Path("/usr/local/share/radiantwave")
UPDATER_PATH = BIN_DIR / "radiantwave-updater.py"
BINARY_PATH = BIN_DIR / "radiantwave"
SCRIPTS_DIR = BIN_DIR / "scripts"
INSTALL_HELPER = BIN_DIR / "radiantwave-installer.sh"

# Version tracking
VERSION_FILE = SHARE_DIR / "VERSION"

# Logging
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
        return None
    
    try:
        with open(VERSION_FILE, 'r') as f:
            version = f.read().strip()
            logger.log(f"Current version: {version}")
            return version
    except IOError as e:
        logger.error(f"Could not read VERSION file: {e}")
        return None


def check_for_update(current_version):
    """Check if update is available on server"""
    logger.log("Checking for updates...")
    
    try:
        url = f"{UPDATE_SERVER}/VERSION"
        with urllib.request.urlopen(url, timeout=10) as response:
            remote_version = response.read().decode('utf-8').strip()
            logger.log(f"Remote version: {remote_version}")
            
            if current_version is None:
                logger.log("Fresh install, will download")
                return remote_version
            
            if remote_version != current_version:
                logger.log(f"Update available: {current_version} -> {remote_version}")
                return remote_version
            else:
                logger.log("Already up to date")
                return None
                
    except urllib.error.URLError as e:
        logger.error(f"Could not check for updates: {e}")
        return None
    except Exception as e:
        logger.error(f"Unexpected error checking for updates: {e}")
        return None


def download_file(url, dest_path):
    """Download file from URL to destination path"""
    logger.log(f"Downloading {url}...")
    
    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            with open(dest_path, 'wb') as f:
                shutil.copyfileobj(response, f)
        
        logger.log(f"Downloaded to {dest_path}")
        return True
        
    except urllib.error.URLError as e:
        logger.error(f"Download failed: {e}")
        return False
    except Exception as e:
        logger.error(f"Unexpected error during download: {e}")
        return False


def verify_checksum(file_path, checksum_path):
    """Verify file SHA256 checksum"""
    logger.log(f"Verifying checksum for {file_path.name}...")
    
    try:
        # Read expected checksum
        with open(checksum_path, 'r') as f:
            expected = f.read().strip().split()[0]  # Format: "hash filename"
        
        # Calculate actual checksum
        sha256 = hashlib.sha256()
        with open(file_path, 'rb') as f:
            for chunk in iter(lambda: f.read(4096), b''):
                sha256.update(chunk)
        actual = sha256.hexdigest()
        
        if actual == expected:
            logger.log("✓ Checksum verified")
            return True
        else:
            logger.error(f"Checksum mismatch!")
            logger.error(f"  Expected: {expected}")
            logger.error(f"  Actual:   {actual}")
            return False
            
    except Exception as e:
        logger.error(f"Checksum verification failed: {e}")
        return False


def download_update(version):
    """Download update package and verify"""
    logger.log(f"Downloading update package for version {version}...")
    
    # Create temporary directory for download
    temp_dir = Path(tempfile.mkdtemp(prefix='radiantwave-update-'))
    logger.log(f"Using temporary directory: {temp_dir}")
    
    try:
        # Download tarball
        tarball_name = f"radiantwave-v{version}.tar.xz"
        tarball_path = temp_dir / tarball_name
        tarball_url = f"{UPDATE_SERVER}/{tarball_name}"
        
        if not download_file(tarball_url, tarball_path):
            return None
        
        # Download checksum
        checksum_name = f"{tarball_name}.sha256"
        checksum_path = temp_dir / checksum_name
        checksum_url = f"{UPDATE_SERVER}/{checksum_name}"
        
        if not download_file(checksum_url, checksum_path):
            return None
        
        # Verify checksum
        if not verify_checksum(tarball_path, checksum_path):
            return None
        
        logger.log("✓ Update package downloaded and verified")
        return tarball_path
        
    except Exception as e:
        logger.error(f"Error downloading update: {e}")
        shutil.rmtree(temp_dir, ignore_errors=True)
        return None


def install_update(tarball_path):
    """Extract and install update package using pkexec"""
    logger.log(f"Installing update from {tarball_path}...")
    
    # Verify install helper exists
    if not INSTALL_HELPER.exists():
        logger.error(f"Install helper not found: {INSTALL_HELPER}")
        return False
    
    try:
        # Use pkexec to run install helper with elevated privileges
        logger.log("Running install helper via pkexec...")
        result = subprocess.run(
            ["pkexec", str(INSTALL_HELPER), str(tarball_path)],
            capture_output=True,
            text=True
        )
        
        if result.returncode != 0:
            logger.error(f"Installation failed: {result.stderr}")
            return False
        
        # Log output from install helper
        if result.stdout:
            for line in result.stdout.splitlines():
                logger.log(f"  {line}")
        
        logger.log("✓ Installation complete")
        return True
        
    except subprocess.CalledProcessError as e:
        logger.error(f"Installation failed: {e}")
        return False
    except Exception as e:
        logger.error(f"Unexpected error during installation: {e}")
        return False


def cleanup_temp_files(tarball_path):
    """Remove temporary download directory"""
    if tarball_path:
        temp_dir = tarball_path.parent
        try:
            shutil.rmtree(temp_dir)
            logger.log(f"Cleaned up temporary files")
        except Exception as e:
            logger.log(f"Warning: Could not clean up {temp_dir}: {e}")


def main():
    """Main updater workflow"""
    logger.log("=========================================")
    logger.log("RadiantWave Updater")
    logger.log("=========================================")
    logger.log(f"Running as user: {os.getenv('USER', 'unknown')}")
    logger.log(f"Log file: {LOG_FILE}")
    logger.log("")
    
    # Check current version
    current_version = get_current_version()
    
    # Check for updates
    new_version = check_for_update(current_version)
    
    if new_version is None:
        logger.log("No update needed")
        logger.log("=========================================")
        return 0
    
    # Download update
    tarball_path = download_update(new_version)
    
    if tarball_path is None:
        logger.error("Failed to download update")
        logger.log("=========================================")
        return 1
    
    # Install update
    success = install_update(tarball_path)
    
    # Cleanup
    cleanup_temp_files(tarball_path)
    
    if success:
        logger.log("")
        logger.log("✓ Update installed successfully!")
        logger.log(f"  New version: {new_version}")
        logger.log("=========================================")
        return 0
    else:
        logger.error("Update installation failed")
        logger.log("=========================================")
        return 1


if __name__ == "__main__":
    sys.exit(main())