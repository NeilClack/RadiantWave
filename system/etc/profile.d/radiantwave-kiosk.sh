# RadiantWave kiosk auto-launch on tty1
if [ -z "$WAYLAND_DISPLAY" ] && [ "$(tty)" = "/dev/tty1" ]; then
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export SDL_VIDEODRIVER=wayland
    exec cage -- /usr/local/bin/radiantwave
fi