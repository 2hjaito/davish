#!/usr/bin/env bash
# Fails unless this commit is the release of the given version: wails.json
# carries the number and CHANGELOG.md has its section.
# Usage: scripts/check-release.sh 1.5.2
set -euo pipefail

want="$1"
have=$(sed -n 's/.*"productVersion": *"\([^"]*\)".*/\1/p' wails.json)

if [ "$have" != "$want" ]; then
  echo "wails.json says $have but the tag says $want: tag the release commit" >&2
  exit 1
fi
if ! grep -qF "## [$want]" CHANGELOG.md; then
  echo "CHANGELOG.md has no section for $want" >&2
  exit 1
fi
