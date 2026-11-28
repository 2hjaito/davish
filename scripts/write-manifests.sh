#!/bin/sh
# Writes the Homebrew cask and the Scoop manifest for one release, from that
# release's SHA256SUMS.txt. The release workflow runs it on checkouts of
# daeviz/homebrew-davish and daeviz/scoop-davish and pushes them.
#
#   scripts/write-manifests.sh 1.15.3 SHA256SUMS.txt ../homebrew-davish ../scoop-davish
set -eu

version="${1:?usage: scripts/write-manifests.sh <version> <SHA256SUMS.txt> <tap dir> <bucket dir>}"
sums="${2:?missing SHA256SUMS.txt}"
tap="${3:?missing tap dir}"
bucket="${4:?missing bucket dir}"
case "$version" in
  [0-9]*) ;;
  *) echo "version must start with a digit, not a v: $version" >&2; exit 1 ;;
esac

sum_of() {
  hash=$(awk -v f="$1" '$2 == f || $2 == "*" f { print $1 }' "$sums")
  case "$hash" in
    [0-9a-f]*) [ "${#hash}" -eq 64 ] && { echo "$hash"; return; } ;;
  esac
  echo "no SHA-256 for $1 in $sums" >&2
  exit 1
}

base="https://github.com/2hjaito/davish/releases/download/v$version"
mac="Davish-v$version-macos-universal.zip"
win="Davish-v$version-windows-amd64.zip"
mac_sum=$(sum_of "$mac")
win_sum=$(sum_of "$win")

mkdir -p "$tap/Casks" "$bucket/bucket"

cat > "$tap/Casks/davish.rb" <<CASK
cask "davish" do
  version "$version"
  sha256 "$mac_sum"

  url "https://github.com/2hjaito/davish/releases/download/v#{version}/Davish-v#{version}-macos-universal.zip"
  name "Davish"
  desc "SSH client with an OS-like desktop for managing Linux servers"
  homepage "https://github.com/2hjaito/davish"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on macos: :monterey

  app "Davish.app"

  # The app is not signed yet, so macOS would block the first start. A plain
  # postflight runs on every Homebrew; the newer postflight_steps does not.
  postflight do
    system_command "/usr/bin/xattr",
                   args: ["-dr", "com.apple.quarantine", "#{appdir}/Davish.app"]
  end

  zap trash: [
    "~/Library/Application Support/Davish",
    "~/Library/Caches/Davish",
  ]
end
CASK

cat > "$bucket/bucket/davish.json" <<SCOOP
{
    "version": "$version",
    "description": "SSH client with an OS-like desktop for managing Linux servers. Needs the WebView2 runtime, which Windows 11 has.",
    "homepage": "https://github.com/2hjaito/davish",
    "license": "GPL-3.0-only",
    "url": "$base/$win",
    "hash": "$win_sum",
    "bin": "Davish.exe",
    "shortcuts": [
        [
            "Davish.exe",
            "Davish"
        ]
    ],
    "checkver": "github",
    "autoupdate": {
        "url": "https://github.com/2hjaito/davish/releases/download/v\$version/Davish-v\$version-windows-amd64.zip"
    }
}
SCOOP
