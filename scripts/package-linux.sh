#!/bin/sh
# Packs the Linux build the way it ships: the binary, its icon and the script
# that adds it to the application menu (#305). The release workflow and the CI
# artifact both use it, so the packaging is exercised on every CI run and not
# only when a tag is pushed.
#
# Run from the repository root after `wails build`:
#   scripts/package-linux.sh dist/Davish-linux-amd64.tar.gz
set -eu

dest="${1:?usage: scripts/package-linux.sh <archive.tar.gz>}"
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

cp build/bin/Davish "$stage/Davish"
cp build/appicon.png "$stage/davish.png"
cp build/linux/install-desktop.sh "$stage/install-desktop.sh"
chmod 755 "$stage/Davish" "$stage/install-desktop.sh"

mkdir -p "$(dirname "$dest")"
tar -C "$stage" -czf "$dest" Davish davish.png install-desktop.sh

# Fail here, not on a user's machine, if the archive is not what the docs say.
have=$(tar -tzf "$dest" | sort | tr '\n' ' ')
want="Davish davish.png install-desktop.sh "
if [ "$have" != "$want" ]; then
  echo "unexpected archive contents: $have" >&2
  exit 1
fi
