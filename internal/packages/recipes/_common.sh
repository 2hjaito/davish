# Davish install recipe. Runs as you; sudo is asked only where needed.
set -e
if command -v apt-get >/dev/null 2>&1; then PM=apt
elif command -v dnf >/dev/null 2>&1; then PM=dnf
elif command -v apk >/dev/null 2>&1; then PM=apk
else echo "No apt, dnf or apk on this server." >&2; exit 1; fi
pkg() {
  case $PM in
    apt) sudo apt-get update && sudo DEBIAN_FRONTEND=noninteractive apt-get install -y "$@" ;;
    dnf) sudo dnf install -y "$@" ;;
    apk) sudo apk add "$@" ;;
  esac
}
start() {
  if command -v systemctl >/dev/null 2>&1; then sudo systemctl enable --now "$1"
  elif command -v rc-update >/dev/null 2>&1; then sudo rc-update add "$1" && sudo rc-service "$1" start
  fi
}
