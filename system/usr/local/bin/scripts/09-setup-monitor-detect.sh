#!/usr/bin/env bash
set -euo pipefail

TARGET_USER="localuser"
TARGET_HOME="/home/${TARGET_USER}"
TARGET_DIR="${TARGET_HOME}/.config/hypr/scripts"
SCRIPT_PATH="${TARGET_DIR}/auto-scale.sh"

# Ensure directory exists with correct ownership and permissions
echo "Creating directory: ${TARGET_DIR}"
install -d -m 755 -o "${TARGET_USER}" -g "${TARGET_USER}" "${TARGET_DIR}"

# Write the auto-scale script
cat > "${SCRIPT_PATH}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

# ----- Config -----
RETRY_WINDOW_SEC=15
POLL_INTERVAL_SEC=1.5
STABLE_TICKS=2

requires() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing dependency: $1" >&2
    exit 1
  }
}
requires hyprctl
requires jq

read_active_json() {
  hyprctl monitors -j 2>/dev/null | jq 'map(select(.disabled==false))[0]'
}

decide_scale() {
  local w="$1" h="$2"
  if (( w >= 3840 && h >= 2160 )); then
    echo "2.0"
  else
    echo "1.0"
  fi
}

apply_scale() {
  local name="$1" scale="$2"
  echo "Applying: $name -> scale=$scale"
  hyprctl keyword monitor "$name,preferred,auto,$scale" >/dev/null
}

initial_with_retry() {
  local deadline=$(( $(date +%s) + RETRY_WINDOW_SEC ))
  while :; do
    local m; m="$(read_active_json || true)"
    if [[ -n "${m:-}" && "$m" != "null" ]]; then
      local name w h
      name="$(jq -r '.name' <<<"$m")"
      w="$(jq -r '.width' <<<"$m")"
      h="$(jq -r '.height' <<<"$m")"
      local scale; scale="$(decide_scale "$w" "$h")"
      apply_scale "$name" "$scale"
      echo "$name:$w:$h:$scale"
      return 0
    fi
    if (( $(date +%s) >= deadline )); then
      echo "No active monitor detected within ${RETRY_WINDOW_SEC}s; giving up."
      return 1
    fi
    sleep 0.5
  done
}

watch_loop() {
  local last_applied_key="$1"
  local stable_count=0
  local last_read_key=""

  while :; do
    local m; m="$(read_active_json || true)"
    if [[ -z "${m:-}" || "$m" == "null" ]]; then
      stable_count=0
      last_read_key=""
      sleep "$POLL_INTERVAL_SEC"
      continue
    fi

    local name w h scale read_key
    name="$(jq -r '.name' <<<"$m")"
    w="$(jq -r '.width' <<<"$m")"
    h="$(jq -r '.height' <<<"$m")"
    scale="$(decide_scale "$w" "$h")"
    read_key="$name:$w:$h:$scale"

    if [[ "$read_key" == "$last_read_key" ]]; then
      ((stable_count++))
    else
      stable_count=1
      last_read_key="$read_key"
    fi

    if (( stable_count >= STABLE_TICKS )) && [[ "$read_key" != "$last_applied_key" ]]; then
      apply_scale "$name" "$scale"
      last_applied_key="$read_key"
    fi

    sleep "$POLL_INTERVAL_SEC"
  done
}

main() {
  case "${1:-}" in
    --watch)
      key="$(initial_with_retry || true)"
      watch_loop "${key:-""}"
      ;;
    *)
      initial_with_retry
      ;;
  esac
}

main "$@"
EOF

# Set ownership and permissions
chown "${TARGET_USER}:${TARGET_USER}" "${SCRIPT_PATH}"
chmod 755 "${SCRIPT_PATH}"