#!/bin/sh
# Renders assets/social-preview.png (1280x640, GitHub social preview) from
# scripts/social-preview.html with headless Chrome. Rerun after retaking
# assets/screenshots/desktop.jpg.
set -eu
cd "$(dirname "$0")/.."
CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
"$CHROME" --headless=new --disable-gpu --hide-scrollbars --allow-file-access-from-files \
  --window-size=1280,640 --screenshot="$PWD/assets/social-preview.png" \
  "file://$PWD/scripts/social-preview.html"
