#!/bin/sh
# Renders the README banners from scripts/social-preview.html with headless
# Chrome, at 2x: assets/social-preview.png (English, also GitHub's social
# preview), social-preview-vi.png and social-preview-zh.png. Rerun after
# retaking a desktop screenshot in assets/screenshots.
set -eu
cd "$(dirname "$0")/.."
CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
for lang in en vi zh; do
  out=assets/social-preview.png
  [ "$lang" = en ] || out=assets/social-preview-$lang.png
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars --allow-file-access-from-files \
    --force-device-scale-factor=2 --virtual-time-budget=2000 \
    --window-size=1280,640 --screenshot="$PWD/$out" \
    "file://$PWD/scripts/social-preview.html?lang=$lang"
done
