#!/bin/sh
# Packs the Linux build as a .deb for Debian, Ubuntu and their relatives, so
# `sudo apt install ./Davish-….deb` pulls GTK and WebKitGTK in and adds Davish
# to the application menu. No new tool: dpkg-deb is on the runners.
#
# Run from the repository root after `wails build`:
#   scripts/package-deb.sh 1.15.3 dist/Davish-v1.15.3-linux-amd64.deb
set -eu

version="${1:?usage: scripts/package-deb.sh <version> <out.deb>}"
dest="${2:?usage: scripts/package-deb.sh <version> <out.deb>}"
case "$version" in
  [0-9]*) ;;
  *) echo "version must start with a digit, not a v: $version" >&2; exit 1 ;;
esac

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
# mktemp makes it 0700, and the package root's mode lands on / when installed.
chmod 755 "$stage"

install -d "$stage/DEBIAN" "$stage/usr/lib/davish" "$stage/usr/bin" \
  "$stage/usr/share/applications" "$stage/usr/share/pixmaps"
install -m 755 build/bin/Davish "$stage/usr/lib/davish/Davish"
ln -s ../lib/davish/Davish "$stage/usr/bin/davish"
install -m 644 build/appicon.png "$stage/usr/share/pixmaps/davish.png"

# StartupWMClass is the window's app id, which Wayland matches to this entry
# to find the icon (#305).
cat > "$stage/usr/share/applications/Davish.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Davish
Comment=Manage Linux servers over SSH
Exec=/usr/bin/davish
Icon=davish
StartupWMClass=Davish
Categories=Network;System;
Terminal=false
DESKTOP

size=$(du -sk "$stage/usr" | cut -f1)
cat > "$stage/DEBIAN/control" <<CONTROL
Package: davish
Version: $version
Section: net
Priority: optional
Architecture: amd64
Installed-Size: $size
Depends: libgtk-3-0t64 | libgtk-3-0, libwebkit2gtk-4.1-0
Maintainer: Davish <https://github.com/2hjaito/davish/issues>
Homepage: https://github.com/2hjaito/davish
Description: SSH client with an OS-like desktop for Linux servers
 Terminal, files, monitor, services, Docker and logs over SSH and SFTP.
 Nothing is installed on the server.
CONTROL

mkdir -p "$(dirname "$dest")"
dpkg-deb --root-owner-group --build "$stage" "$dest" >/dev/null

# Fail here, not on a user's machine, if the package is not what the docs say.
have=$(dpkg-deb --contents "$dest" | awk '{print $6}' | sort | tr '\n' ' ')
for want in ./usr/bin/davish ./usr/lib/davish/Davish ./usr/share/applications/Davish.desktop ./usr/share/pixmaps/davish.png; do
  case " $have" in
    *" $want "*) ;;
    *) echo "missing $want in $dest: $have" >&2; exit 1 ;;
  esac
done
