#!/bin/bash
set -euo pipefail

# --- Args + prompt for system/channel ---
valid_system() { [[ "$1" == "home" || "$1" == "commercial" ]]; }
valid_channel() { [[ "$1" == "release" || "$1" == "beta" || "$1" == "dev" ]]; }

SYSTEM_TYPE="${1:-}"
CHANNEL="${2:-}"

while ! valid_system "${SYSTEM_TYPE:-}"; do
  read -rp "Which system type? (home|commercial): " SYSTEM_TYPE
done
while ! valid_channel "${CHANNEL:-}"; do
  read -rp "Which release channel? (release|beta|dev): " CHANNEL
done

echo "--------------------------------------------------"
echo " RadiantWave setup"
echo " System : ${SYSTEM_TYPE}"
echo " Channel: ${CHANNEL}"
echo " (All actions live — no chroot mode)"
echo "--------------------------------------------------"
read -rp "Proceed? (y/N): " CONFIRM
[[ "$CONFIRM" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 1; }

if [[ "$SYSTEM_TYPE" != "home" && "$SYSTEM_TYPE" != "commercial" ]]; then
  echo "Usage: $0 [home|commercial]"
  exit 1
fi

cd /tmp

declare -A STATUS
run_step() {
  local name="$1"; shift
  echo -e "\n[*] Starting: $name..."
  if "$@"; then
    STATUS["$name"]="OK"; echo "[✓] $name completed successfully."
  else
    STATUS["$name"]="FAILED"; echo "[✗] $name failed, continuing..."
  fi
}

# --- Minimal deps ---
run_step "Install minimal dependencies" bash -c '
  pacman -Syu --noconfirm --needed \
    wayland \
    sdl2 sdl2_ttf sdl2_mixer \
    cronie nvim vi \
    intel-ucode amd-ucode
'

# --- Initramfs / UKI / LUKS+TPM ---
run_step "Configure mkinitcpio" \
  sed -i 's/^HOOKS=.*/HOOKS=(base systemd autodetect microcode modconf kms keyboard sd-vconsole block sd-encrypt filesystems fsck)/' /etc/mkinitcpio.conf

run_step "Write kernel cmdline" bash -c '
  CRYPT_DEV="/dev/nvme0n1p2"
  LUKS_UUID="$(blkid -s UUID -o value "$CRYPT_DEV")"
  if [[ -z "${LUKS_UUID:-}" ]]; then
    echo "[WARN] Could not read LUKS UUID from $CRYPT_DEV"; exit 1
  fi
  CMDLINE="rd.luks.name=${LUKS_UUID}=cryptroot rd.luks.options=${LUKS_UUID}=tpm2-device=auto root=/dev/mapper/cryptroot rw quiet loglevel=3 systemd.show_status=auto"
  echo "$CMDLINE" | tee /etc/kernel/cmdline >/dev/null
'

run_step "Fix mkinitcpio preset for UKI" bash -c '
  sed -i "/^default_options=/d" /etc/mkinitcpio.d/linux.preset
  echo "default_options=\"--splash /dev/null\"" >> /etc/mkinitcpio.d/linux.preset
'

run_step "Bind TPM2 to LUKS (if TPM present)" bash -c '
  if [[ -e /dev/tpmrm0 ]]; then
    systemd-cryptenroll --tpm2-device=auto --tpm2-pcrs=0+7 /dev/nvme0n1p2
  else
    echo "[WARN] /dev/tpmrm0 not found; deferring TPM2 enrollment to first boot"
  fi
'

run_step "Verify TPM2 enrollment (non-fatal if deferred)" bash -c '
  cryptsetup luksDump /dev/nvme0n1p2 | grep -A4 -n "Token:" || {
    echo "[WARN] TPM2 token not visible yet (ok if deferred)"; exit 0
  }
'

run_step "Rebuild initramfs + UKI" bash -c 'mkinitcpio -P'

run_step "Set systemd-boot timeout" bash -c '
  sed -i "/^timeout/d" /boot/loader/loader.conf
  echo "timeout 1" | tee -a /boot/loader/loader.conf >/dev/null
'

# --- Pull RadiantWave payload ---
run_step "Download VERSION file" \
  bash -c 'curl -fsSLO "https://repository.radiantwavetech.com/basic/'"$CHANNEL"'/'"$SYSTEM_TYPE"'/VERSION"'

VERSION="$(cat VERSION || echo 'unknown')"
echo "[*] Latest RadiantWave version is $VERSION"

BASE="radiantwave-${SYSTEM_TYPE}-${VERSION}.tar.xz"
SUM="${BASE}.sha256"

run_step "Download checksum" \
  bash -c 'curl -fsSLO "https://repository.radiantwavetech.com/basic/'"$CHANNEL"'/'"$SYSTEM_TYPE"'/'"$SUM"'"'
run_step "Download tarball" \
  bash -c 'curl -# -fsSLO "https://repository.radiantwavetech.com/basic/'"$CHANNEL"'/'"$SYSTEM_TYPE"'/'"$BASE"'"'
run_step "Verify checksum" sha256sum -c "$SUM"
run_step "Extract package" tar --no-same-owner -xJf "$BASE" -C /
run_step "Cleanup downloads" rm -f VERSION "$BASE" "$SUM"

# --- Users / groups / offline linger ---
run_step "Ensure localuser exists" bash -c '
  id -u localuser &>/dev/null || useradd -m -s /bin/bash localuser
  usermod -aG video,input,audio,render,wheel localuser || true
'
run_step "Disable password for localuser" passwd -d localuser || true

run_step "Enable linger for localuser" bash -c '
  install -d -m 755 /var/lib/systemd/linger
  : > /var/lib/systemd/linger/localuser
'

# --- Copy Hyprland config to localuser ---
run_step "Install Hyprland config for localuser" bash -c '
  mkdir -p /home/localuser/.config/hypr
  chmod 700 /home/localuser/.config
  chmod 700 /home/localuser/.config/hypr
  chown -R localuser:localuser /home/localuser/.config
  if [[ -f /usr/local/share/radiantwave/hyprland.conf ]]; then
    cat /usr/local/share/radiantwave/hyprland.conf > /home/localuser/.config/hypr/hyprland.conf
    chown localuser:localuser /home/localuser/.config/hypr/hyprland.conf
    chmod 600 /home/localuser/.config/hypr/hyprland.conf
  else
    echo "[WARN] /usr/local/share/radiantwave/hyprland.conf not found"
  fi
'

# --- NEW: Run post-install hooks (from extracted payload) ---
run_step "Run post-install hooks" bash -c '
  set -euo pipefail
  POST_INSTALL_DIR="/usr/local/share/radiantwave/post_install"
  export SYSTEM_TYPE="'"$SYSTEM_TYPE"'"
  export CHANNEL="'"$CHANNEL"'"
  export VERSION="'"$VERSION"'"

  if [[ -d "$POST_INSTALL_DIR" ]]; then
    shopt -s nullglob
    # sort -V respects numeric prefixes like 10-foo.sh, 20-bar.sh
    mapfile -t scripts < <(printf "%s\n" "$POST_INSTALL_DIR"/*.sh | sort -V)
    if (( ${#scripts[@]} == 0 )); then
      echo "[INFO] No post-install scripts found in $POST_INSTALL_DIR"
    else
      for script in "${scripts[@]}"; do
        [[ -f "$script" ]] || continue
        echo "[INFO] Running post-install: $script"
        bash "$script" || echo "[WARN] Post-install script failed: $script"
      done
    fi
    shopt -u nullglob
  else
    echo "[INFO] No post-install directory at ${POST_INSTALL_DIR}; skipping."
  fi
'

# --- Autologin via SDDM ---
run_step "Fix stray/duplicate Hyprland session file" bash -c '
  set -e
  uwsm="/usr/share/wayland-sessions/hyprland-uwsm.desktop"
  plain="/usr/share/wayland-sessions/hyprland.desktop"
  if [[ -f "$uwsm" && -f "$plain" ]]; then
    if ! pacman -Qo "$plain" &>/dev/null; then
      echo "[INFO] Removing unowned $plain"
      rm -f "$plain"
    fi
  fi
'

run_step "Write SDDM autologin config" bash -c '
  install -d -m 755 /etc/sddm.conf.d
  SESSION_FILE="hyprland-uwsm.desktop"
  [[ -f /usr/share/wayland-sessions/hyprland.desktop ]] && SESSION_FILE="hyprland.desktop"
  cat > /etc/sddm.conf <<EOF
[Autologin]
User=localuser
Session=${SESSION_FILE}
Relogin=true

[General]
DisplayServer=wayland
EOF
  chmod 644 /etc/sddm.conf
'

# --- Updater cron (system-wide) ---
run_step "Add updater cron jobs to /etc/crontab" bash -c '
  sed -i "/radiantwave-updater/d" /etc/crontab
  cat >> /etc/crontab << "EOF"

# RadiantWave updater jobs
@reboot root sleep 10 && /usr/local/bin/radiantwave-updater
0 0 * * * root /usr/local/bin/radiantwave-updater
EOF
  systemctl reload cronie.service || systemctl restart cronie.service || true
'

# --- Enable and start services ---
run_step "Enable + start core services" bash -c '
  systemctl enable cronie.service || true
  systemctl disable getty@tty1.service || true
  systemctl enable  getty@tty2.service || true
  systemctl enable  sddm.service || true
  systemctl start   cronie.service || true
  systemctl stop    getty@tty1.service || true
  systemctl start   getty@tty2.service || true
  systemctl restart sddm.service || systemctl start sddm.service || true
'

# --- Verify kiosk binary ---
run_step "Verify kiosk binary exists" bash -c '
  if [[ ! -x /usr/local/bin/radiantwave ]]; then
    echo "[ERROR] /usr/local/bin/radiantwave is missing or not executable."; exit 1
  fi
'

# --- Final Report ---
echo -e "\n===== Installation Summary ====="
for step in "${!STATUS[@]}"; do
  printf "%-45s : %s\n" "$step" "${STATUS[$step]}"
done
echo "================================"

echo -e "\n===== Service Health Check ====="
systemctl is-active sddm.service cronie.service getty@tty2.service || true
echo "================================"
