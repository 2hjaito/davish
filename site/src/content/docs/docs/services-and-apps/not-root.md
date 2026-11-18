---
title: "Not root?"
description: "Logged in as a user who may use sudo, the apps run what the server refuses as root: Services, Docker , Cron, Logs, Files and Editor, Disk, Nginx, Security and Database."
sidebar:
  order: 10
---

Logged in as a user who may use `sudo`, the apps run what the server refuses as root: Services, Docker (outside the `docker` group), Cron, Logs, Files and Editor, Disk, Nginx, Security and Database. Davish shows the exact command and asks for the sudo password once per server and session: the apps of that server then share it until it goes unused for 15 minutes (as sudo itself does; 5 minutes, an hour or the whole session in **Settings → App lock**), you disconnect, the connection drops or you lock the app. Then it is asked again. It is kept in memory only (never on disk, in the keychain or in a log) and hands it to sudo on stdin, never on a command line. **Forget sudo password** in the server menu wipes it. When sudo needs no password nothing is asked. A user who may not use sudo is not asked at all. What could lock you out of the server (turning ufw on, password logins off) and paths other users could swap stay in the terminal.
