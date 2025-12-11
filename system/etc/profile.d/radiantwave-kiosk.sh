# RadiantWave kiosk auto-launch on tty1
if [ -z "$WAYLAND_DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export SDL_VIDEODRIVER=wayland
    
    # Check for updates before launching
    /usr/bin/python3 /usr/local/bin/radiantwave-updater.py || true
    
    exec cage -- /usr/local/bin/radiantwave
fi