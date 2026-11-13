# Changelog

Notable changes to Davish, in English. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/). Versions before 1.7.0 are only in the Vietnamese [CHANGELOG.md](CHANGELOG.md).

## [Unreleased]

### Added
- Settings → Dock: put the Dock on the bottom, left, right or top edge (windows and notifications keep clear; autohide comes back from that edge), and a transparent Dock with only the icons.
- All servers: **Map** on a connected server's card shows where requests go: each nginx site as a chain `domain → nginx :443 → :3000 → container (stack)`, static sites with their folder, and ports open to the outside without a proxy. Alerts come first: a database port open to the Internet, an upstream nobody listens on, a certificate expiring within 14 days, each with a button to the app that fixes it. Below, what runs and the disks. Read only. With four servers or more, a name filter and servers with alerts first.
- Launchpad: a **Recent** row at the top of All, with the five apps opened last (from the Dock, Spotlight or Launchpad), remembered across restarts.
- Monitor → Overview: the machine's details at the bottom of the page (the separate Info section and the Load · Uptime · Swap · GPU strip are gone: load sits under the CPU figure, swap in use and GPUs join the Hardware card, GPUs only when there are some), four cards, System (boot time, time zone, systemd, who is logged in), Hardware (cores, threads, sockets), Disks (NVMe/SSD/HDD/virtual drives with model, size, partitions and mounts; which mount holds Docker's data and whose volumes and bind mounts, through `docker info` and `docker inspect`, logged in the command log) and Network (LAN, VPN and Docker interfaces with addresses, gateway, DNS). Read only, through `lsblk`, `ip` and `/proc`.
- Docker: containers as cards, one per compose project (or lone container), with a technology logo when Davish knows the image (n8n, WordPress, Postgres…), how many of its containers run, its image, published ports that open in the Browser, CPU and RAM; a red edge when one keeps restarting, and a click shows and controls its containers. A compose project's card takes the project's own favicon, or the framework named in `package.json`, `requirements.txt`, `composer.json`, `Gemfile`, `go.mod`… (Next, Nest, Express, Django, FastAPI, Laravel…), read from the project folder over SFTP; nothing runs on the server. A Cards/Table switch in the toolbar; the table by default, and Davish remembers your choice.

## [1.13.5] - 2026-11-12

### Changed
- The sudo password is now forgotten after 15 minutes unused, per server, as sudo itself does; Davish used to keep it until you disconnected or locked the app. Pick 5 minutes, an hour or the whole session in Settings → App lock → Forget the sudo password. Checking sudo's state does not count as a use.

## [1.13.4] - 2026-11-11

### Fixed
- Docker: logged in as a user outside the `docker` group, the container, image and volume lists and start, stop, restart, remove and new network/volume no longer stop at "needs root": Davish runs the same command again through `sudo`. With no password needed it just runs; otherwise the app offers "View with sudo…" and asks once per session, as Services and Nginx do (#117). Lines typed into Terminal (run, pull, exec, compose) are unchanged.
- Cron: logged in as a user who may use sudo, Cron also shows root's crontab, and adds, edits, removes, enables and disables its jobs through `sudo crontab -u root -`. The old crontab is backed up in `/root` first, and the confirmation shows the exact command and warns that the job runs as root. Unreadable files in `/etc/crontab` and `/etc/cron.d`, and a job's recent runs, are read through sudo too. When a password is needed, "View with sudo…" asks once per session (#117). A user without sudo does not see root's crontab. Files in `/etc/cron.d` are still edited in Editor.
- Logs: a log you may not read (the system journal outside the `adm`/`systemd-journal` groups, root's files, docker outside the `docker` group) is followed again through `sudo`: right away when no password is needed, else "View with sudo…" asks once per session (#117). The password only goes on the stream's stdin, never on a command line or in the audit log; when sudo turns it down it is forgotten and asked again.
- Files and Editor: logged in as a user who is not root, deleting root's files and editing read-only files no longer only fall back to Terminal (#117). Delete: when the server refuses, Davish runs the confirmed `rm` again through `sudo`. It never deletes as root `/`, the system folders (`/usr`, `/boot`, `/proc`…), home folders, or the files needed to log in (`/etc/passwd`, `/etc/shadow`, `/etc/sudoers`, `/etc/ssh`…). Editor: "Edit with sudo…"; saving shows the exact command, then writes through `sudo` (a temporary file `mktemp` creates, owner and permissions kept, a backup), and refuses if the file changed since it was opened. The sudo password is asked once per session; the prompt's Terminal button types `sudoedit`. Deleting in Disk also goes through `sudo`, within Disk's own refusals.
- Disk: logged in as a user who is not root, the cleanups that need root (the journal, old logs in `/var/log`, apt, dnf, crash dumps, `/tmp`, snap, container logs) run through `sudo` once confirmed instead of only being typed into Terminal (#117); the Terminal button stays. With the session's sudo password, Davish measures the disk and what can be cleaned as root; without it, "Measure with sudo…" shows the exact command first. Emptying a root's file runs again through `sudo` too, within Disk's refusals. The user's own caches (`~/.cache`, npm) always run as the user.
- Security: blocking and unblocking with ufw, and banning and unbanning with fail2ban, no longer need passwordless sudo: Davish runs them through `sudo`, asking for the sudo password once per session as the other apps do (#117). What needs root to read (`lastb`, the ufw and fail2ban status, logins in the journal) is read as root when it can be; when a password is needed, "View with sudo…" shows the exact command. Turning ufw on and password logins off, which could lock you out, stay in Terminal only. Users who may not use sudo are not offered the prompt.
- Database: when only root may connect to MariaDB or PostgreSQL, Davish no longer needs passwordless sudo: listing, creating, dropping, dumps and restores run through `sudo`, asking for the sudo password once per session (#117), with "View with sudo…". The password is never on a command line: SQL follows it on stdin, dumps are still written by the user into `~/db-backups`, and a restore hands sudo the password line before the dump. Users who may not use sudo are not offered the prompt.

## [1.13.3] - 2026-11-11

### Added
- Monitor on Windows servers: CPU per core, memory and swap, drives, network, disk throughput, GPUs, processes, system info and service/container alerts, read through Windows PowerShell 5.1 and CIM over SSH. Nothing is installed on the server. Load average, I/O wait and steal time do not exist on Windows and are hidden. Works whether the server's SSH shell is cmd.exe or PowerShell.

### Fixed
- Monitor on a Windows or macOS server no longer waits for the first reading for ever. Davish now detects the server's operating system right after it connects, in the background, so connecting is never held up by it; Monitor says it does not support that system yet (or that the server returned no readings) instead of retrying in silence. Linux servers are unchanged.
- Terminal on a Windows server opens the plain login shell instead of sending it a Linux-only start line.

### Changed
- Apps that need a POSIX shell (Services, Docker, Git, Logs, Cron, Nginx and the rest) are hidden from the Dock, Spotlight and the Go menu on a Windows server, and shown greyed out in Launchpad with the reason. Nothing changes for macOS servers except Monitor, which now says it does not support them yet instead of waiting.

## [1.13.2] - 2026-11-11

### Added
- Spanish (Spain) interface: pick Español in Settings → Appearance → Language; the first time, Davish follows a system set to Spanish. The translation is a first draft, to be reviewed by a native speaker (#124).

### Security
- Built with Go 1.27.4, which fixes an HTTP/1 server crash in net/http (GO-2026-6859).

## [1.13.1] - 2026-11-11

### Fixed
- Command confirmation: the Run in Terminal button sits in the bottom row with a label instead of a small icon beside the command; long commands wrap at spaces, so an IP address or a path is not cut in two.
- Settings: the language is picked from a list instead of a row of buttons, so more languages fit.

## [1.13.0] - 2026-11-10

### Added
- Database: MongoDB (installed on the server). Lists databases with their collections and size, creates a database with a user and a random password, deletes, dumps and restores (`mongodump`/`mongorestore`), and opens a tunnel for Compass. With authentication on, log in once with fields or a connection string; Davish tries it first and only then keeps it in this computer's keychain (like the SSH password), with "Forget login". The connection string only travels on stdin, never on a command line or in the command log.
- Database: SQLite. Add a file by path, with Find, or by right-clicking a `.db`/`.sqlite`/`.sqlite3` file in Files → "Open in Database". Shows tables and rows (opened read-only), backs up safely while an app writes, and restores with a line typed into Terminal.
- Database: Find (read-only) looks for the database servers running and the SQLite files on the server.
- Database: "How does Davish connect?" explains where commands run and how passwords are kept.

### Fixed
- Nginx: logged in as a user who is not root, enabling, disabling, removing, creating and editing a site, testing and reloading no longer only fall back to Terminal: Davish runs the confirmed command again through `sudo`, asking for the sudo password once per session as Services does (#117). `certbot` stays in Terminal, since it may ask questions.

## [1.12.3] - 2026-11-10

### Fixed
- Sudo: "a password is required" is recognized on servers not set to English (sudo runs with `LC_ALL=C`); two actions needing the password at once now share one prompt instead of the first one vanishing; servers with `requiretty` (older CentOS/RHEL) fall back to Terminal instead of failing; a user in none of the sudo, wheel or admin groups is warned before typing the password, with Terminal offered first.

## [1.12.2] - 2026-11-10

### Fixed
- Services: logged in as a user who is not root, start, stop, restart, removing a service and its journal are no longer refused (#76, #113). Davish runs them again through `sudo`: right away when sudo needs no password, else after asking for the sudo password once per session. The password is kept in memory only, never on disk, in the keychain or in a log, reaches sudo on stdin, and is wiped when you disconnect, lock the app or choose File → Forget sudo password. The prompt shows the exact `sudo …` command and can run it in Terminal instead.

## [1.12.1] - 2026-11-10

### Added
- Security: tick several addresses to block (or unblock) them at once. One dialog lists the exact command for each; they run in turn, and any that fail or need a sudo password are typed into Terminal.

### Changed
- Security: Failed logins and Blocked are full-width tables (IP, tries, users tried, last try, status, Block/Unblock on the row) with selectable text. Clicking a row opens its detail below instead of in a pane on the right.

## [1.12.0] - 2026-11-10

### Added
- Security: block or unblock an IP with ufw, ban or unban it with fail2ban, right in the app. The dialog shows the exact command and runs it once you confirm (`sudo -n`, when sudo needs no password); a small Terminal button beside the command types it into Terminal instead. When sudo needs a password, Terminal opens as before. A warning when the address has an open login session (it may be you) or ufw is off.
- Security: copy an IP from its detail pane or from the tables; IPs in tables can be selected.

### Changed
- Security: sections on the left, data on the right (Overview, Failed logins, Blocked, Users tried, Successful logins, Open sessions, Firewall, Password login), with a filter by IP or user. Clicking an overview tile opens its section, such as Blocked listing every blocked address and whether ufw or fail2ban blocks it.
- Security: ufw and fail2ban status is read through `sudo -n` when it can be, so blocked addresses show without logging in as root.

### Fixed
- The Database and Deploy windows had no title bar (no close, minimize or maximize buttons, nothing to drag them by) and a see-through background; Database's Dump, Restore and Delete looked like plain browser buttons.

## [1.11.0] - 2026-11-10

### Added
- Security: a 7-day overview at the top of the app: failed logins, unknown users, addresses, banned, successful logins; failed logins per hour (hover or use ←/→ to read each hour); the addresses and user names tried most; open sessions (`who`).
- Saved commands: keep commands with a name for one server or every server, type one into Terminal in a click (it does not run on its own), find them in Spotlight; a warning when a command looks like it holds a password, token or key. Command log: every line can be typed into Terminal again or saved as a command.
- All servers window (server menu → All servers…, Launchpad, Spotlight): a card per server with its connection, CPU, memory, `/` disk, load, uptime, network, alert count and latest alert; connect or switch to a server from its card. Readings are collected only while the window is open.
- Launchpad groups apps (Files & terminal, Monitoring, Web & containers, Security & access, Development, System) with a tab per group; the arrow keys move across groups; pinned apps always show their mark; a "Restore the default Dock" button (click twice).

### Fixed
- Spotlight: a result's name (such as "Help: Services") is no longer cut to a few letters when its description is long.

## [1.10.1] - 2026-11-09

### Changed
- Menu bar: each reading is a gauge open at the bottom, with the number inside and its name below; each one has a fixed width, so a changing value no longer shifts the others. The menu bar is a little taller (34px).

## [1.10.0] - 2026-11-09

### Added
- Menu bar: each reading has a small ring and its name; hover it for the value, details and the last 15 minutes as a chart. Load, Disk I/O and GPU (only when the server has an NVIDIA GPU) are new; Swap and Uptime can be turned on in Settings → Menu bar, which turns each reading on or off.

## [1.9.0] - 2026-11-09

### Added
- Simplified Chinese interface (简体中文): pick it in Settings → Appearance → Language; on a Chinese system Davish picks it the first time. Release notes show in English.

### Changed
- Release notes come in English and Vietnamese; the update dialog shows the ones in your interface language.

## [1.8.2] - 2026-11-09

### Fixed
- Updates: when GitHub fails a download (like the 504 outage on 2026-11-09), Davish tries again and then downloads through the GitHub API instead of giving up.
- A failed update now says so at the top of the dialog, naming the step that failed (checking or downloading), with Try again, Download manually and Details. Details show the error code and the updater's log lines, with Copy for a bug report and a button to open the log folder.

## [1.8.1] - 2026-11-09

### Fixed
- Windows and Linux: the Davish window was always maximized and could not be moved to another monitor or minimized. The menu bar now ends with Minimize, Maximize/Restore and Close; drag the empty part of the menu bar (the stats and clock) to move the window, double-click it to maximize or restore, and drag the window's edges to resize it (#93).

## [1.8.0] - 2026-11-09

### Added
- Settings → Dock: pick the Dock size (Small, Medium, Large) and turn on Automatically hide. The Dock slides below the screen; move the pointer to the bottom edge to bring it back, and maximized or snapped windows use that space.

### Fixed
- Disk: the folder, Large files and Clean up lists did not scroll, so the rows at the bottom were cut off; they scroll now.

## [1.7.0] - 2026-11-08

### Added
- Docker, Nginx, Git and Database: when the server lacks the program, the app says so and shows the install command for the server's Linux (apt, dnf, yum, apk), with enabling the service. "Run in Terminal" types the command for you to read and press Enter. Docker is installed from the distro's packages and adds you to the `docker` group (the app says this group is as good as root, with a button to reconnect). Once installed, coming back to the window opens the app (#79). An SSH user without sudo is told to ask an administrator, with no command.
- Disk: opening a disk scans it once and shows an overview: a bar split into measured folders, deleted files still held open, what could not be measured and free space. What could not be measured comes with the reason: folders the SSH user cannot read (with a `sudo du` command typed into a Terminal) and deleted files a process still holds. When sudo needs no password, Davish measures as root and sees everything.
- Disk: folders as a tree that opens in place, and a Large files tab (from 50 MB). Delete a folder or file, or empty a log file, after seeing the exact command; from 1 GB, or for a folder with contents, type its name. System folders, Docker and database data and home folders cannot be deleted from here.
- Clean up: Docker is split into unused images, stopped containers, build cache, volumes (with a warning that they hold data) and container logs over 10 MB; plus crash dumps, temporary files older than 7 days, old snap revisions, `~/.cache` and the npm cache.

### Changed
- On first start the Dock pins only the everyday apps: Terminal, Files, Monitor, Services, Docker, Logs, Editor, Settings. Other apps open from Launchpad or Spotlight and can be pinned as before. A Dock you already pinned or unpinned stays as it is.

### Fixed
- Git: a server without git used to show "No repository"; it now says git is missing and offers to install it.
- Long commands in the "needs root" box no longer break in the middle of a word.
- Docker clean up used to run `docker system prune -f`, which only removes dangling images, so the disk stayed full while the app reported many GB freed. Each kind now has its own command that matches the size shown.
