# Greetd Auto-Login Setup for Ubuntu Server 24

## Overview
This guide configures greetd for automatic login of the `kiosk` user on Ubuntu Server 24, bypassing password authentication while keeping the user account password-protected for other purposes.

## Installation

```bash
# Install greetd
sudo apt update
sudo apt install greetd

# Disable other display managers if running
sudo systemctl disable gdm3 lightdm sddm
```

## Configuration

Edit `/etc/greetd/config.toml`:

```toml
[terminal]
vt = 1

[default_session]
command = "agreety --cmd /bin/bash"
user = "kiosk"

[initial_session]
command = "/bin/bash"
user = "kiosk"
```

## Auto-Start Application

Create `/home/kiosk/.bash_profile`:

```bash
#!/bin/bash
# Auto-start RadiantWave on login
if [ -z "$DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
    exec /usr/local/bin/radiantwave
fi
```

Make it executable:
```bash
chmod +x /home/kiosk/.bash_profile
chown kiosk:kiosk /home/kiosk/.bash_profile
```

## Enable Service

```bash
sudo systemctl enable greetd
sudo systemctl start greetd
```

## Verification

Reboot the system. The `kiosk` user should auto-login on tty1 and launch RadiantWave automatically.

## Notes

- The `kiosk` user password remains active for SSH, sudo, and manual logins on other TTYs
- Auto-login only applies to tty1 (the primary console)
- If RadiantWave crashes, the user remains logged in at a bash prompt
