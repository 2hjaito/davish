---
title: Install
description: Install Davish on macOS, Windows or Linux.
sidebar:
  order: 1
---

Prebuilt apps for macOS (universal), Windows and Linux are attached to each [release](https://github.com/2hjaito/davish/releases), or use the buttons on the [home page](/#download). Davish is free and open source (GPL-3.0).

## Package managers

```sh
brew install --cask daeviz/davish/davish      # macOS
scoop bucket add davish https://github.com/daeviz/scoop-davish
scoop install davish                          # Windows
sudo apt install ./Davish-…-linux-amd64.deb   # Debian, Ubuntu: download the .deb first
```

Homebrew 7 refuses a cask from a third-party tap you have only tapped. The one-line command above trusts it as it installs; if you ran `brew tap daeviz/davish` first, also run `brew trust daeviz/davish`. The tap and the bucket follow each release. Homebrew also removes macOS's block on the unsigned app. The `.deb` is owned by apt, so install the next version the same way.

## macOS

macOS 12 or later, Apple silicon or Intel. Open `Davish-…-macos-universal.pkg` to install into Applications.

Davish is not code-signed yet, so macOS blocks it the first time:

1. Click **Done**.
2. Open **System Settings → Privacy & Security**.
3. Click **Open Anyway** next to the line about Davish (again for `Davish.app` if it is blocked).

Or run `xattr -dr com.apple.quarantine` on the downloaded file.

## Windows

Windows 10 or later, 64-bit. Run `Davish-…-windows-amd64-setup.exe`; it installs the WebView2 runtime if it is missing. SmartScreen may warn about an unknown publisher: choose **More info → Run anyway**.

The `.zip` is a portable build: unzip it and run `Davish.exe`, nothing to install.

## Linux

x86-64 with GTK 3 and WebKitGTK 4.1 (on Debian and Ubuntu: `libgtk-3-0` and `libwebkit2gtk-4.1-0`). Unpack `Davish-…-linux-amd64.tar.gz` and run `Davish`.

The archive also holds `davish.png` and `install-desktop.sh`. Run `./install-desktop.sh` once to add Davish to your application menu with its icon. This matters on Wayland (KDE, GNOME), where a window's icon comes from a desktop entry and not from the program itself: without it, the window and the taskbar show no icon. Start Davish from the menu afterwards. The script writes one file, `~/.local/share/applications/Davish.desktop`, points it at the folder it is run from (run it again if you move the folder), and `./install-desktop.sh --remove` takes it away. It needs no root.

## Updates

Davish checks for new versions on its own. When one is out it shows what changed, with **Update**, **Later** and **Skip this version**; it downloads the update, checks its signature and restarts into it. See [Updates](/docs/the-desktop/updates/).
