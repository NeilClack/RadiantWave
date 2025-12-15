# RadiantWave kiosk auto-launch on tty1
if [ -z "$WAYLAND_DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
  export XDG_RUNTIME_DIR=/run/user/$(id -u)
  export SDL_VIDEODRIVER=wayland

  # Check for updates before launching
  /usr/bin/python3 /usr/local/bin/radiantwave-updater.py || true

  # Set all sinks to full volume
  for sink in $(wpctl status | grep -oP '^\s+\K\d+(?=\.)' | head -10); do
    wpctl set-volume "$sink" 1.0 2>/dev/null
  done

  exec cage -- /usr/local/bin/radiantwave
fi

