# Davish features

**English** · [Tiếng Việt](FEATURES.vi.md) · [Back to README](README.md)

Every app in Davish, what it does and what it runs on the server. The interface is in Vietnamese; screenshots show demo data from the browser preview.

## Terminal

xterm.js with tabs, copy/paste, GPU rendering, and an optional powerline prompt injected into bash at login, with nothing written on the server.

<img src="assets/screenshots/desktop.jpg" alt="Terminal, Monitor and Files" width="100%">

## Files

Finder-style browser over SFTP: upload, download, drag and drop from your OS, rename, delete, compress, and copy between servers through your machine.

## Editor

CodeMirror 6 with highlighting for nginx, YAML, JSON, shell, env and config files. Atomic saves that keep mode and owner, backups under `/etc`, and a warning if the file changed on the server.

## Logs

Follow a unit's journal, a file or a container. Filter, pause, highlight errors, and click a file path to open it in the Editor at that line.

<img src="assets/screenshots/logs.jpg" alt="Logs with clickable file paths" width="100%">

## Monitor

CPU per core, memory and swap, disks, network per interface, NVIDIA GPUs, processes. RAM and disk alerts you can tune.

## Services

systemd units by state, their status and journal; start, stop and restart. Remove a unit you no longer use: an admin's unit is deleted, a package's (such as an old `certbot`) is masked with the timer that runs it.

<img src="assets/screenshots/services.jpg" alt="Services" width="100%">

## Docker

Containers with live CPU and memory, logs with stderr marked, inspect, start/stop/restart, restart-loop alerts. Images with the containers using them, their layers and config; run one in a terminal or remove the unused.

<table><tr>
  <td><img src="assets/screenshots/docker.jpg" alt="Docker with a container in a restart loop"></td>
  <td><img src="assets/screenshots/docker-images.jpg" alt="An image and its layers"></td>
</tr></table>

## Cron

Your crontab and the system ones, schedules in plain words, next runs in the server's time zone, safe edits with a backup.

<img src="assets/screenshots/cron.jpg" alt="Cron jobs with schedules in plain words" width="100%">

## Nginx

Sites, server blocks and upstreams; turn sites on or off, `nginx -t` before every reload, errors that open at the right line; TLS expiry alerts. Delete a site, and its Let's Encrypt certificate if no other site uses it.

<table><tr>
  <td><img src="assets/screenshots/nginx.jpg" alt="Nginx site with TLS certificate"></td>
  <td><img src="assets/screenshots/remove.jpg" alt="Deleting an nginx site with its certificate"></td>
</tr></table>

## Packages

What apt, dnf, yum or apk installed and can upgrade, security fixes first, with an alert when you connect. The runtimes and server programs it finds (nginx, MariaDB, Redis, Docker, Node.js with nvm, PHP with php-fpm, Python…), each with its version, config files and service; also in Spotlight. Installing and upgrading type the `sudo` command into a terminal for you.

<table><tr>
  <td><img src="assets/screenshots/packages.jpg" alt="Pending updates, security fixes first"></td>
  <td><img src="assets/screenshots/packages-apps.jpg" alt="Programs found on the server"></td>
</tr></table>

## Git

Finds the repositories in your home, `/var/www`, `/srv` and `/opt`, or folders you add. Branch, upstream and how far ahead or behind, changed files with their diffs, and a commit graph of every branch with merges and tags. Fetch, `pull --ff-only` (refused while the tree is dirty) and switch branches after confirming the exact command; `npm ci` / yarn / pnpm from the lockfile in a terminal. A rejected key leads to creating a deploy key.

<table><tr>
  <td><img src="assets/screenshots/git.jpg" alt="The commit graph of every branch"></td>
  <td><img src="assets/screenshots/git-changes.jpg" alt="Changed files and a diff"></td>
</tr></table>

## Ports & tunnels

Every TCP port the server listens on, with its process or container. Open one in Davish's own browser window, or forward it to `127.0.0.1` on your machine over the SSH connection you already have, open it in your browser or point a database tool at it, even as a domain name for nginx sites. Nothing is opened to the outside.

<img src="assets/screenshots/ports.jpg" alt="Listening ports and a tunnel" width="100%">

## Browser

Type `localhost:3000` and see the web app running on the server, inside a Davish window: back, forward, reload, redirects and websockets (hot reload) stay on the server, logins keep working. It goes over SSH through a proxy on a random `127.0.0.1` port, so nothing clashes with what runs on your machine, and the page lives at its own origin, never able to reach Davish.

<img src="assets/screenshots/browser.jpg" alt="A web app on the server, opened inside Davish" width="100%">

## Disk

What fills each disk, one folder at a time and deeper on a double click, without scanning the whole tree. Cleanups with the space they free (Docker, the journal, rotated logs, package caches), each showing its exact command; a full-disk alert opens it at the right mount.

<table><tr>
  <td><img src="assets/screenshots/disk.jpg" alt="What fills the disk, one folder at a time"></td>
  <td><img src="assets/screenshots/disk-clean.jpg" alt="Cleanups with the space they free"></td>
</tr></table>

## Security

Who logs in over SSH and who keeps trying: failed attempts grouped by address with the names they tried, recent logins, ufw and fail2ban. Block an address or turn password logins off with a `sudo` command typed into a terminal, checked by `sshd -t` before the reload.

<img src="assets/screenshots/security.jpg" alt="Failed logins grouped by address" width="100%">

## SSH keys

The server's `authorized_keys` with the key Davish logs in with marked; add or remove keys with a backup. Create a deploy key on the server and test it against GitHub or GitLab, checked against their published host key. Generate a key on your machine, put it on the server and switch to it.

<img src="assets/screenshots/keys.jpg" alt="The server's authorized_keys" width="100%">

## ~/.ssh/config and ProxyJump

Import hosts from `~/.ssh/config` in one go, keys and `ProxyJump` included; Davish only reads the file. Servers behind a bastion connect through it, each hop with its own host key check, and the whole chain reconnects on its own.

## Spotlight & Launchpad

A real menu bar (File, View, Go, Window, Help and the focused app's own commands) with every shortcut listed under `⌘/`, and a `?` on each app explaining what it does and which commands it runs. `⌘K` (or the magnifier in the menu bar) to open an app, a service, a container, a folder, a `/path`, or `> a command` typed into a terminal. Launchpad (`⌘⇧A`) shows every app and lets you pin them to the Dock.

<table><tr>
  <td><img src="assets/screenshots/spotlight.jpg" alt="Spotlight"></td>
  <td><img src="assets/screenshots/launchpad.jpg" alt="Launchpad"></td>
</tr></table>

## Menu bar and help

A real menu bar (File, View, Go, Window, Help and the focused app's own commands), every shortcut under `⌘/`, a `?` on each app explaining what it does and which commands it runs on the server, a first-run tour and a demo mode with sample data.

<table><tr>
  <td><img src="assets/screenshots/menu.jpg" alt="The Go menu with its shortcuts"></td>
  <td><img src="assets/screenshots/help.jpg" alt="An app's help"></td>
</tr></table>

## App lock

Touch ID or a password of its own covers everything when you step away. Forgot the password? Resetting it removes the saved server passwords first, so it never opens a way into a server. Windows open, close and minimize into the Dock with a genie effect.

<table><tr>
  <td><img src="assets/screenshots/lock.jpg" alt="The lock screen"></td>
  <td><img src="assets/screenshots/genie.jpg" alt="The genie effect"></td>
</tr></table>
