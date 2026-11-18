---
title: What the server needs
description: OpenSSH with SFTP is enough; a few apps use tools the server may already have.
sidebar:
  order: 3
---

Davish is agentless: nothing is installed on, copied to or left behind on the server. Everything runs from your computer over SSH.

| Feature | Requirement |
|---|---|
| Everything | OpenSSH server with the SFTP subsystem |
| Services, Logs (journal) | systemd; to read system logs, a user in the `adm` or `systemd-journal` group |
| Docker | the `docker` CLI; a user in the `docker` group (Davish offers the install command when it is missing) |
| Cron | `crontab` |
| Nginx | nginx with `sites-available` / `sites-enabled` or `conf.d` |
| Database | the engine's own client: `mysql`, `psql`, `mongosh` or `sqlite3` |

Actions that need root (stopping a system service, editing a root-owned file) show the exact command first, then run through `sudo` with the password asked once per session, or go to a terminal with `sudo` typed in. Nothing runs as root silently, and what could lock you out of the server stays in the terminal.

Windows servers work too, on a best-effort basis: Monitor reads them, and the rest depends on what the app needs.
