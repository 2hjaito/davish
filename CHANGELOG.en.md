# Changelog

Notable changes to Davish, in English. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/). Versions before 1.7.0 are only in the Vietnamese [CHANGELOG.md](CHANGELOG.md).

## [Unreleased]

### Fixed
- Command confirmation: the Run in Terminal button sits in the bottom row with a label instead of a small icon beside the command; long commands wrap at spaces, so an IP address or a path is not cut in two.

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
