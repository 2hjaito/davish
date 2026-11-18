---
title: Install
description: Install Davish on macOS, Windows or Linux.
sidebar:
  order: 1
---

Prebuilt apps for macOS (universal), Windows and Linux are attached to each [release](https://github.com/2hjaito/davish/releases), or use the buttons on the [home page](/#download). Davish is free and open source (GPL-3.0).

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

## Updates

Davish checks for new versions on its own. When one is out it shows what changed, with **Update**, **Later** and **Skip this version**; it downloads the update, checks its signature and restarts into it. See [Updates](/docs/the-desktop/updates/).
