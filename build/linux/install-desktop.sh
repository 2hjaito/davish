#!/bin/sh
# Adds Davish to the application menu and gives its window an icon on Wayland
# (KDE, GNOME), where the icon comes from a desktop entry that matches the
# window's app id ("Davish"), not from the window itself.
#
#   ./install-desktop.sh           add the entry for the Davish in this folder
#   ./install-desktop.sh --remove  take it away again
#
# It writes one file, Davish.desktop, in your own applications folder
# (~/.local/share/applications). Run it again after moving this folder.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
apps="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
entry="$apps/Davish.desktop"

if [ "${1:-}" = "--remove" ]; then
  rm -f "$entry"
  echo "Removed $entry"
  exit 0
fi

if [ ! -x "$here/Davish" ] || [ ! -f "$here/davish.png" ]; then
  echo "Run this from the folder that holds Davish and davish.png." >&2
  exit 1
fi

# A desktop entry quotes a path in double quotes; these cannot go inside them.
case "$here" in
  *'"'* | *'\'* | *'$'* | *'`'* | *'%'*)
    echo "The folder name has a character a desktop entry cannot hold: $here" >&2
    exit 1 ;;
esac

mkdir -p "$apps"
cat > "$entry" <<EOF
[Desktop Entry]
Type=Application
Name=Davish
Comment=Manage Linux servers over SSH
Exec="$here/Davish"
Icon=$here/davish.png
StartupWMClass=Davish
Categories=Network;System;
Terminal=false
EOF

# Optional: some desktops refresh their menu on their own.
command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database "$apps" >/dev/null 2>&1 || true

echo "Added $entry"
echo "Davish shows in the application menu now; start it from there so its window gets the icon."
