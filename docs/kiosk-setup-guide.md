# RadiantWave Kiosk Setup Guide - Ubuntu Server 24

Complete setup guide for a fresh Ubuntu Server 24 installation configured as a RadiantWave kiosk.

## Initial System Setup

### 1. Create Kiosk User

```bash
sudo adduser kiosk
sudo usermod -aG sudo kiosk
```

### 2. Install Dependencies

```bash
sudo apt update
sudo apt install -y \
    libsdl2-2.0-0 \
    libsdl2-ttf-2.0-0 \
    libsdl2-mixer-2.0-0 \
    libegl1-mesa \
    libgles2-mesa \
    libgbm1 \
    libinput10 \
    libxkbcommon0 \
    dbus \
    network-manager
```

### 3. Install Cage (Wayland Compositor)

```bash
sudo apt install -y cage
```

### 4. Install Greetd

```bash
sudo apt install -y greetd
```

### 5. Configure PAM for Greetd

Create `/etc/pam.d/greetd`:

```bash
sudo tee /etc/pam.d/greetd <<EOF
#%PAM-1.0
auth       required   pam_unix.so
auth       required   pam_env.so
account    required   pam_unix.so
password   required   pam_unix.so
session    required   pam_unix.so
session    required   pam_limits.so
EOF
```

### 6. Configure Greetd Auto-Login

Create `/etc/greetd/config.toml`:

```bash
sudo tee /etc/greetd/config.toml <<EOF
[terminal]
vt = 1

[initial_session]
command = "cage -s -- /usr/local/bin/radiantwave"
user = "kiosk"

[default_session]
command = "cage -s -- /usr/local/bin/radiantwave"
user = "kiosk"
EOF
```

### 7. Disable Network Wait Service

Speed up boot by disabling network wait:

```bash
sudo systemctl disable systemd-networkd-wait-online.service
sudo systemctl mask systemd-networkd-wait-online.service
```

## Install RadiantWave

### Option 1: Via APT Repository (Recommended)

```bash
# Add repository
echo "deb [trusted=yes] https://repository.radiantwavetech.com/release ./" | \
  sudo tee /etc/apt/sources.list.d/radiantwave.list

# Install
sudo apt update
sudo apt install radiantwave
```

### Option 2: Manual Installation

```bash
# Download and install .deb package
wget https://repository.radiantwavetech.com/radiantwave_X.X.X_amd64.deb
sudo dpkg -i radiantwave_X.X.X_amd64.deb
sudo apt install -f
```

## Enable and Start Greetd

```bash
sudo systemctl enable greetd
sudo systemctl start greetd
```

## Verify Setup

Check greetd status:
```bash
sudo systemctl status greetd
```

Check logs if issues occur:
```bash
journalctl -xeu greetd
```

## Reboot

```bash
sudo reboot
```

The system should auto-login as `kiosk` and launch RadiantWave in fullscreen via cage.

## Troubleshooting

### Greetd fails to start
- Check PAM config: `cat /etc/pam.d/greetd`
- Check greetd config syntax: `cat /etc/greetd/config.toml`
- View detailed logs: `journalctl -xeu greetd`

### Black screen after login
- Verify cage is installed: `which cage`
- Verify radiantwave exists: `ls -l /usr/local/bin/radiantwave`
- Check cage can access GPU: `ls -l /dev/dri/`
- Add kiosk to video group: `sudo usermod -aG video kiosk`

### Network not connecting
- Check NetworkManager status: `systemctl status NetworkManager`
- Use nmtui for WiFi setup: `sudo nmtui`
- RadiantWave will show WiFi setup page if no connection detected

### Auto-updates not working
- Verify polkit rules: `ls -l /etc/polkit-1/rules.d/99-radiantwave-updater.rules`
- Check updater script: `ls -l /usr/local/bin/radiantwave-updater.py`
- Test manual update: `sudo apt update && sudo apt list --upgradable`

## Security Notes

- The kiosk user keeps their password for SSH and sudo access
- Auto-login only applies to tty1 (the primary console)
- Other TTYs (Ctrl+Alt+F2-F6) still require password
- Polkit rules restrict kiosk user to only RadiantWave package updates

## Remote Management

### SSH Access
```bash
ssh kiosk@<kiosk-ip>
```

### View RadiantWave Logs
```bash
# From SSH session
sudo journalctl -u greetd -f
```

### Restart RadiantWave
```bash
sudo systemctl restart greetd
```

### Update RadiantWave
```bash
sudo apt update
sudo apt upgrade radiantwave
```
