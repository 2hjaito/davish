# Changelog

Notable changes to Davish, in English. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and versions follow [SemVer](https://semver.org/). Versions before 1.7.0 are only in the Vietnamese [CHANGELOG.md](CHANGELOG.md).

## [Unreleased]

### Added
- **Install with Homebrew and Scoop**: `brew install --cask daeviz/davish/davish` on macOS; `scoop bucket add davish https://github.com/daeviz/scoop-davish`, then `scoop install davish` on Windows. The release workflow updates both repositories after each release (it needs the `TAP_TOKEN` secret).
- **Linux `.deb` package** (Debian, Ubuntu and relatives, amd64): `sudo apt install ./Davish-…-linux-amd64.deb` pulls in GTK and WebKitGTK and adds Davish to the application menu. Built by the release workflow beside the `.tar.gz`; CI builds one on every run.
- **PuTTY `.ppk` keys**: choose a `.ppk` file (from puttygen, format versions 2 and 3; RSA, ECDSA and Ed25519, with or without a passphrase) as a server's key directly, with no export to OpenSSH first. Davish reads it itself and checks the file's MAC, which tells a wrong passphrase from a damaged file, and writes nothing to disk. Version 1 files and Argon2d still say how to convert them in PuTTYgen. Approach learned from [motty](https://github.com/t8y2/motty) (Apache-2.0); the code is our own.
- **GPU processes, like nvtop**: Monitor → GPU now lists the processes on each NVIDIA GPU below the cards: PID, user, GPU, type, utilization, GPU memory and its share of the card, CPU, host memory and the command, sorted by any column. It runs `nvidia-smi` and reads `/proc` for those processes every 3 seconds, only while the GPU tab is open. Where the driver has no `pmon` only compute processes show, without utilization; Windows in WDDM mode shows no GPU memory per process.
- German interface: pick Deutsch in Settings → General → Language; the first time, Davish follows a system set to German.
- **Korean interface** (#327, from @wbg1991): pick 한국어 in Settings → General → Language; the first time, Davish follows a system set to Korean. The cron description now also reads right in languages that put the time after its word.
- **French interface**: pick Français in Settings → General → Language; the first time, Davish follows a system set to French (France, Canada, Belgium, Switzerland). The first draft was made with AI, so wrong or clumsy words are welcome in [Discussion #323](https://github.com/2hjaito/davish/discussions/323).
- **Reorder the Dock**: drag a pinned icon along the Dock to move it, as on macOS. The other icons step aside to show where it lands, and the new order is kept when you let go. It stays between Launchpad and the open apps that are not kept in the Dock, Escape while dragging puts it back, and a click still opens the app. Pins for apps another server's system has keep their places.
- **Hold a widget to edit the widgets**: hold a desktop widget still for half a second and widget editing turns on, as on a phone's home screen, without going through View → Edit widgets. Letting go then opens nothing, and moving without letting go drags the widget as before. A short click still opens the widget's app.
- **Servers reached through a ProxyCommand** (#320, from @nullmax): a server behind a WebSocket relay, Cloudflare Access, AWS SSM or a bastion that only `ProxyCommand` reaches can now be added, or imported from `~/.ssh/config` with its command. Davish runs the command on this computer through your shell, as ssh does, with `%h`, `%n`, `%p`, `%r` and `%%` expanded. A command you type in the server dialog runs at once; one that came from an import is shown first, with the exact line it will run, and Davish asks once and again if the command changes. A host or login with a character the shell would read is refused before it reaches the command. Not for use with a jump server or a control socket.
- **Open with sudo** (#177): a folder or file your SSH user cannot read can now be opened. Right-click it in Files and choose Open with sudo…, or press the same button next to the permission error, in Files for a folder and in Editor for a file. Davish shows the exact read-only command first (`find` to list, `stat` and `head` to read) and runs it through sudo, asking for the sudo password once per session if needed. A file opened this way is read-only; Edit with sudo… saves it as root as before, and deleting still asks first.

### Fixed
- Monitor → GPU now reads AMD and Intel GPUs, not only NVIDIA (#313): a server with a Radeon or an Intel iGPU used to show "No NVIDIA GPU found". On Linux the cards come from the kernel (`/sys/class/drm`: utilization, GPU memory, temperature and power for AMD; the name for Intel, whose driver reports no load there), and the process table from the DRM descriptors each process holds open (`/proc/PID/fdinfo`, the source nvtop uses): GPU time since the last reading and GPU memory. Without root only your own processes are visible. On Windows servers every vendor is read from Windows' own GPU counters (engine load, memory per card and per process); no temperature or power there. Nothing is installed on the server. Not yet tried on a real AMD or Intel GPU; thanks to @tian2992 for the NVIDIA test and the Intel screenshots.
- SSH agent that does not sign (#332, 1Password on Windows): when the agent refuses the login or closes the connection while waiting for you to confirm the key, Davish said "ssh handshake failed" (unreachable), which sends you looking at the network. It now says the agent did not sign, and points at the agent's confirmation prompt and its own log.
- 1Password on Windows showed no prompt for Davish, so every login through its SSH agent waited a minute and failed (#332). 1Password names the app that asks for a key after the parent of the process holding its pipe, and Davish opened from the Start menu or Explorer has only Explorer above it ("process tree is empty" in 1Password's log). Davish now opens the agent's pipe through a copy of itself that runs without a window, and 1Password asks "Allow Davish". Tried on Windows 10 with 1Password from the Microsoft Store. Found on the way: an agent that lists no keys is asked once more a second later (1Password lists none the first time while it is locked), and the "no key found" error now says to unlock 1Password or Bitwarden before pointing at Windows' own agent; Disconnect while the agent waits for you ends that login, where it used to stay waiting behind the next one; trusting a new server's host key no longer shows "Something went wrong" when the connection after it fails, which has its own error dialog.
- Dates followed one fixed order whatever the language (#326, from @wbg1991): the Git app always wrote commit dates in the Vietnamese format, and the menu bar clock, file dates and cron run times always wrote day before month. Each language now writes its own.
- SSH agent that asks you first (Bitwarden): Davish closed the connection after 10 seconds, under a "Confirm SSH key usage" prompt that nobody had answered yet, and logged nothing (#168). While the agent is signing the login, Davish now waits for you, up to a minute (or `DAVISH_AGENT_WAIT` if that is longer), and gives the server a fresh moment to answer once it has signed. The log says when the agent starts signing, how long it took and how it ended, and a handshake that fails for another reason is logged too. If the minute passes, the error now says the agent did not sign in time and to look for its confirmation window.
- Linux: the archive now holds the icon and `install-desktop.sh`, which adds Davish to the application menu. On Wayland (KDE, GNOME) a window's icon comes from a desktop entry, so before this the window showed no icon (#305). Not tried on a Wayland desktop yet.
- Control socket: a server reached through an OpenSSH control socket no longer shows "Connection lost" about 15 seconds after it connects (#301). Davish's keepalive request is not forwarded by OpenSSH's mux proxy, which dropped the connection that sent it. Davish now sends no keepalives over a control socket: the master looks after the server (`ServerAliveInterval`), and a master that exits closes the socket, which Davish still notices. Found by @PlayerFann14.
- Settings: a setting with a long explanation and a row of buttons beside it squeezed the text into a narrow column (the "Dragging windows" row ran to 13 lines in Vietnamese). Dragging windows, When Davish opens, Scrollback, Alert hold and Sudo forget now put the buttons under the text, like the Performance level above (#304).
- Docker: **‹ Containers** on a container's page goes back to the table again; the click did nothing (#310). The app menu read the page's container after going back had dropped it, and the error stopped the page from closing. Items an app adds to the menu bar now also leave it when their part of the window closes; before, they stayed behind.
- All servers → Map: the map had no margin, so its cards touched the edge of the sidebar and the window. It now has the same margin as the list of servers. The title was shown twice and there were two refresh buttons; the title is now "name map" in the window's top bar, and that bar's refresh button reads the map again (before, it only reloaded the list of servers).
- Deleting a server now names the other servers that go through it as a jump host and will connect directly afterwards, so that is seen before you confirm. The rest of the audit after #296 found nothing else to change in the script shortcuts, the updater (signed checksums, no downgrades) or the app lock reset.

## [1.15.2] - 2026-11-26

### Added
- **A picture as the desktop wallpaper** (M53): Settings → Appearance has **Picture** under the wallpaper color: choose a PNG, JPEG or WebP (under 50 MB) from your computer. It is scaled down to 2560 px and kept on this computer, never uploaded, and **Dim picture** darkens it (up to 70%) so icons and text stay readable. **Settings → Performance → Wallpaper picture** switches it off on its own: Beautiful and Balanced show it, Light does not, and without a picture nothing changes or costs anything. On the development Mac a picture added about 30 MB of memory and no measurable CPU or dropped frames (render bench, 3 runs: `scripts/bench.sh --wallpaper`, which anyone can run on their own machine).
- **Dragging windows** (#262): Settings → Performance has a new setting for how a window follows the pointer while you drag it. **Live** is what it was. **Simple** moves the window too, but turns the glass off until you let go. **Outline** moves only an outline and puts the window there when you let go, for computers that paint the window on the CPU, where dragging a big translucent window is slow. Outline has a light tint inside, which **Fill the outline** turns off for the lightest drag. It is Live unless you change it, and the performance levels are unchanged.
- **System logo beside a server's name**: the cards and the list in All servers and the server menu show the logo of the system each server runs (Ubuntu, Debian, Fedora, Arch, Windows, macOS and more). Davish reads it from `/etc/os-release` when the server connects and remembers it; a server whose system is not known yet, or has no logo of its own, shows a plain server glyph
- **Choose what opens when Davish starts** (#273, #281): Settings → General has **When Davish opens**: **All servers** shows the list of servers, **Last server** opens and connects to the one you used last (what Davish always did), and **Nothing** opens the last server's workspace without connecting, for servers behind a VPN or in production. A new install opens on All servers; if you already have servers, you keep Last server.
- **Turn off password login from Security**: Password login in Security is now a switch instead of a command to type in a terminal. Turning it off checks that your key logs in over a new connection, backs up the sshd settings, adds a drop-in file, checks it with `sshd -t` and `sshd -T`, reloads sshd and checks the key again, and undoes the change if the key no longer logs in. It is not offered when a `Match` block or `AuthenticationMethods` would make it lock people out or not apply, or when Davish logs in with a password. The dialog shows the whole script first, and the same switch turns password login back on
- **Duplicate a server** (#257): the copy icon on a card or a row in All servers opens the add dialog filled in from that server, with the name as `name (copy)`, so only what differs is typed. Nothing is created until you save. Passwords, key passphrases and the control socket are not copied.
- **Connect through an OpenSSH control socket** (#240): for logins that only work through OpenSSH, such as an agent that issues certificates on demand (authentik), the server dialog has an **Advanced: use an ssh control socket** field. Start a master yourself (`ssh -M -S <path> -fN user@host`) and give Davish its path: it attaches to that connection, so OpenSSH does the login and Davish never sees your credentials. It replaces the authentication of that server, and works on Linux and macOS only. Tried against authentik by @PlayerFann14.

### Changed
- Disk → Clean up: before a cleanup runs, the confirmation lists the files it would remove (the largest 200, with sizes) for old logs, container logs, core dumps, temporary files and `~/.cache`. The list comes from the same selection as the delete command, read-only. When it is read without root, it says so. The old text claiming that logs being written are not touched was wrong and is gone (#296).
- With a picture as the wallpaper, the menu bar now follows the picture's brightness instead of the theme: light text on smoky glass over a dark picture, dark text on light glass over a bright one, so its labels, icons and clock stay readable (as on macOS). It is worked out once, when the picture is chosen, from the strip at the top and the dim level, and only while the glass is see-through.
- The readings in the menu bar (CPU, RAM, Disk, network) are bigger: the ring, its number and its name grow by about a tenth, so they are easier to read (#288).
- The log now says when the SSH agent could not be reached, and why (for example a Windows pipe that is busy or not there), so a support log can tell "no agent" from "an agent that would not open", and how long the agent took to answer. An agent that waits for you to unlock or confirm, such as Bitwarden's, may need more than the 5 seconds Davish waits: start Davish with `DAVISH_AGENT_WAIT=60` (seconds, 1 to 300) to wait longer (#168).

### Fixed
- Security: Block and Ban refuse an address that would cut you off: `0.0.0.0/0` and other very wide ranges, loopback, and the address Davish connects from (the server tells it through `SSH_CLIENT`). Run in several servers: the typed confirmation now also covers `apt remove/purge`, `find -delete`, `xargs`, `mv`, `cp`, `sed -i`, `tee`, `docker compose down`, `git reset --hard`, `ufw disable` and similar. Database: dropping a system database (`mysql`, `postgres`, `admin`…) is refused, and restoring a MongoDB archive says it replaces each collection of the same name.
- Disk: cleanups were rechecked after #296. Temporary files now need to be neither read nor changed for 7 days and skip a service's private `/tmp`. The Docker stopped-container and unused-volume cleanups list what they would remove. Texts that promised too much now say what is lost: images you built yourself cannot be pulled again, container logs lose their history, `~/.cache` can hold large downloads. Nginx: deleting a site's Let's Encrypt certificate now says Davish cannot see a mail server or other program that may use it.
- All servers: the list scrolls again when its servers are taller than the window, in both the cards and the table. In 1.15.1 it was cut off at the window's edge and the servers below could not be reached (#256).
- A PuTTY `.ppk` key used as a server's private key now gets a clear message with how to convert it (PuTTYgen → Conversions → Export OpenSSH key), instead of a generic "cannot read the private key" (#264). Davish still cannot read `.ppk` itself.
- Disk: the "Old rotated logs" cleanup no longer deletes files like PCP's live archive volumes (`20261125.00.10.0`). It matched any name ending in `.0`-`.9`; now the number must follow a letter, as in `syslog.1` or `auth.log.2` (#296).

## [1.15.1] - 2026-11-24

### Added
- **Server tags** (#243): give a server a short tag such as `local`, `staging`, `prod` or your own (up to 24 characters) in the add/edit server dialog; the tag takes the server's existing color. It shows next to the server name in the menu bar, the server menu, All servers and each window's title bar, with a thin colored line under the menu bar, so you can tell which environment you are in. The server menu and All servers can filter by tag once there are two or more. Servers without a tag look the same as before.
- **Server groups** (#243): the add/edit server dialog has a **Group** field (one level, up to 40 characters, suggesting the groups already in use). **All servers** now uses the same layout as Docker: a sidebar filters by connection state (All, Connected, Failed, Not connected), by group and by tag, each with a count. Servers of one group sit under a folder heading that folds (it remembers how you left it, and shows how many are up and how many alerts), in both the cards and the table. The search matches groups too. With no group on any server, the list looks as before. The server menu in the menu bar is grouped and folds too, and `⌘1…⌘9` number the servers in the order shown (group A is 1, 2, 3, group B is 4, 5, 6); without groups nothing changes.

### Changed
- In All servers, the tag filter moves from a row of buttons to the sidebar, Add, Import and Run a command become icons in the toolbar, and Cards/Table are two icons like in Docker.
- The status dot in the server menu is one color now, the connection state, with the word under each name gone (hover the dot for it). A server's own color, which a tag chip carries, shows as a thin line at the row's edge when it has no tag. The `⌘1…⌘9` hints line up in one column, and a folder's count sits next to its name.

### Fixed
- A server whose login contains `@` can now be saved and imported from `~/.ssh/config`: a Mac joined to a corporate directory (where `whoami` prints `jane.doe@example.com`), SSSD or FreeIPA `user@realm` logins. The name may be up to 64 characters (#249, thanks @andre0xFF).

## [1.15.0] - 2026-11-23

### Added
- The **Widgets** app (Launchpad, Spotlight, Dock if you pin it): a gallery of the widgets with a live preview, how many of each are on the desktop of the server in use, and buttons to add one small or medium. Its icon is three white cards on green.
- A **Docker** widget: how many containers run, and which ones fail their healthcheck or keep restarting (small, or medium with their names). It reads `docker ps` once a minute while Davish is in front and again when you come back to it, for each server that has one, and never asks for a sudo password; a server that has no Docker or wants sudo is left alone until you come back. It is the only widget that runs a command. Not offered on a Windows server.
- A **Services** widget: how many services run as a coloured bar, and which ones failed or keep restarting, small or medium. It reads `systemctl list-units` once a minute while Davish is in front and again when you come back, with the same care as the Docker widget (never a sudo password, a server without systemd is left alone). Docker and Services share one reading code.
- A **Disk** widget: how full the fullest disk is, with a ring (small), or every disk with a bar and its size (medium). It turns amber and red as the disk alert threshold in Settings → Alerts is near and passed, so it is red when the alert would be. It reads the same collector as Monitor: no command of its own.
- Widgets on the desktop: **View → Add widget: Monitor** puts CPU, or CPU, RAM and network with a 15 minute chart, on the desktop of the server in use; Drag one to move it, as you drag an icon; **View → Edit widgets** also makes them small or medium and removes them. Each kind has its own color and icon, so they can be told apart. Each server has its own. Off in the Light performance preset, or on its own in Settings → Performance → Advanced.
- Settings per app. Terminal: color schemes (Davish, Phosphor, Solarized, Dracula), a font of its own, cursor shape and blinking, a solid background and the length of the scrollback (1,000, 5,000 or 20,000 lines). Editor: font size, wrapping, line numbers, marks for spaces and tabs, and what the Tab key inserts. Files: show or hide names that start with a dot.

### Changed
- Monitor stops asking the server for the process list while its window is not on screen (minimized, on another server, or Davish hidden).

### Fixed
- Widgets start right under the menu bar and sit the same distance from the right edge and from the bar, whatever the window width. Before, a row was kept empty at the top and the gap on the right changed with the window.
- Terminal: the grey of stack traces, timestamps and hints (ANSI bright black) is readable on the dark glass; the Davish color scheme sets every ANSI color now.
- Terminal: the progress of a deploy or a script now sits in the title bar, beside the tabs, instead of floating over the output, where it hid the right end of the first lines.
- Shortcut labels read `Ctrl+Shift+U` and `Ctrl+Alt+L` outside the Mac, Touch ID is only mentioned on macOS, the ⋮ buttons in the Git and Docker toolbars match the buttons beside them, and a few layout slips (the Map button in All servers, Disk's legend) are gone.
- Browser: a page that redirects with a relative path (such as `/` to `/ui/`, or adding a trailing `/`) no longer ends on "this address only opens inside Davish" (403).

## [1.14.11] - 2026-11-21

### Added
- Appearance: 8 wallpaper colors (adds pink, yellow, sky and gray) and a color wheel first that opens the system color picker for any color. A new color glides in over half a second.
- Performance: **Check again** times frames with Beautiful on and says which preset suits this machine (Beautiful, Balanced or Light), with the frame rate and a button to use it.

### Changed
- Account: the name sits under the picture and a click edits it in place. Hovering the picture shows a camera; a click chooses a new one or removes it.
- The account picture is kept at 256 px, so it is sharp on Retina screens (pick it again once for the sharp copy).

### Fixed
- Icons and text in windows no longer look soft after a minimize and restore, or when a window sits on a fractional pixel.
- The menu bar, Dock and desktop icons follow a new wallpaper color right away, without waiting or switching windows.
- A long name no longer spills out of the Settings sidebar, the lock screen or the sleep screen.

## [1.14.10] - 2026-11-21

### Fixed
- Lists without a detail pane take the whole width of the window: Security shows all five cards on one row, and the Docker table no longer cuts off its Actions column.

## [1.14.9] - 2026-11-20

### Added
- Git: an account at the top of the sidebar, with a search over the repositories. Its page shows the SSH user, git's name and email, whether HTTPS tokens are stored, the keys in `~/.ssh` and the GitHub/GitLab account the server's keys sign in to (read only; each host is asked on its own, and one the server never reaches is left out).
- Git: repositories are grouped by the user or org of their remote, the account the server signs in as first and those without a remote last.
- Settings: an **Account** pane at the top of the sidebar with your name, picture and the app lock password (moved from Security). The name and picture stay on this computer and show on the lock and sleep screens.

### Changed
- Git: Clone runs with one button when the server has an accepted key or the repository is public, then opens the new repository. Only one that needs a token is typed into a terminal, as before.
- Settings: the Davish card used to open General, so the two were the same; it is now **About Davish** at the bottom of the sidebar, with the version, license and updates.
- The sleep screen shows your picture and name, as on macOS; with the app lock on, waking from it asks for the password.

## [1.14.8] - 2026-11-19

### Changed
- Settings is rebuilt in the style of macOS System Settings: a sidebar with a search field and a colored icon for each pane (General, Appearance, Performance, Menu bar, Dock, Terminal, Editor, Alerts, Security), the chosen pane's grouped cards on the right, and back and forward between the panes you visited. The theme is picked from previews, and menu bar readings each get their own switch. Language and Updates move to General, App lock and sudo to Security. Davish remembers the last pane open.

### Fixed
- All servers now opens with `⌘O` (Ctrl+O). On Windows the webview keeps Ctrl+0 for its zoom and never passes it on, so the `⌘0` from 1.14.7 did nothing there. Thanks @HungNth (#209).
- Docker: the containers table now shows each compose project's icon like the cards do, including projects that run locally built images (they used to get a plain letter).

## [1.14.7] - 2026-11-18

### Fixed
- All servers now opens with `⌘0` (Ctrl+0), next to `⌘1`…`⌘9` for one server each. 1.14.3 had given it `⌘⇧S`, which Services already had, so only one of them ran; `⌘⇧S` is back to Services. Thanks @HungNth (#202).
- Editor: the search and replace fields, buttons and options use the interface's text size instead of about 9px, readable without zooming in. Thanks @HungNth (#202).

## [1.14.6] - 2026-11-18

### Added
- Database: the SQLite row grid sorts (click a column header: ascending, descending, off) and filters by the text a column contains. Davish builds the SQL from the table's real column names; there is still no SQL box, and it still reads 100 rows at a time.

### Fixed
- Run on several servers: pressing Stop right as a command started could miss it and hold the run until the command finished on its own. Davish now sends the stop signal to the command on the server, and a server that still keeps the session going no longer holds the job past 5 seconds.

### Changed
- Linux: the window is now drawn on the GPU (WebKitGTK's own default), so scrolling, moving windows and frosted glass are smooth instead of painted by the CPU. Davish used to keep Wails' workaround that turned GPU compositing off for everyone. Tested on Intel and AMD graphics; on the proprietary NVIDIA driver, where the workaround came from, the GPU stays off by default, and `DAVISH_WEBVIEW_GPU=always` turns it on (with `WEBKIT_DISABLE_DMABUF_RENDERER=1`, unless you set it yourself). `DAVISH_WEBVIEW_GPU=never` turns it off on any machine, and the app log says which policy was used and why. Thanks to @tian2992 for spotting it and testing on Intel, AMD and NVIDIA (#181, #159, #185).

## [1.14.5] - 2026-11-18

### Added
- Docker: a cleaner container table. Each row has the tech logo, the name over its image, a colored state dot, ports as `8080→80` that open on click, a Created column, and restart, stop/start, delete and more buttons always in view. A compose project's containers hang off it with a tree line. The ID moved to the detail page.

### Fixed
- Windows: the close, minimize and maximize buttons are redrawn like macOS's (a darker ring, solid glyphs that stay crisp at 12px, their own colors when pressed and when the window is in the background). Only fills change, so hover and press are lighter than before, with no `filter`.

## [1.14.4] - 2026-11-18

### Added
- Database: click a SQLite table to read its rows in a grid, read-only, 100 to a page. Long text is cut at 200 characters and blobs show only their size, and nothing runs until you click, so the app stays light.

## [1.14.3] - 2026-11-17

### Added
- All servers: a compact **List** view next to the cards (remembered), add, import, edit and **disconnect** servers right from the window, and **⌘⇧S** (Ctrl+Shift+S) to open it. The server menu also has a disconnect button next to each connected server. Suggested by @HungNth and @tian2992 (#178).
- Database: when the server has no `sqlite3`, the SQLite section shows the install command like MariaDB and Postgres do, instead of pointing to Packages. Each engine and the SQLite section sit in their own card.

### Fixed
- Monitor: the line and fill of the metric charts (Memory, CPU, GPU, Network, Disk and the Overview sparklines) no longer spill past the left edge of the card. Thanks @thanhtrung5763 (#186).
- macOS and Linux: the commands shown in confirm dialogs, inspectors and error dialogs can be selected and copied.


## [1.14.2] - 2026-11-17

### Fixed
- Linux: the Davish window shows the app icon in the taskbar and Alt+Tab instead of the default X11 one (#181).

## [1.14.1] - 2026-11-17

### Added
- Settings → Performance: four presets, **Auto**, **Beautiful**, **Balanced** and **Light**, replace the Reduce glass and Reduce motion switches. Each one sets the frosted blur, the window shadow, solid fills, the wallpaper blobs and motion; **Advanced** sets each on its own, and on Linux adds two WebKitGTK workarounds that apply after a restart. **Auto** keeps the full look, times a second and a half of frames at launch without blocking anything, and offers Light (Use Light / Later) when the machine renders in software or drops frames; **Check again** measures once more. Thanks to @tian2992 (#176).

### Fixed
- Terminal: with GPU acceleration on, the last key typed or the screen a program leaves on exit (such as `btop`) no longer waits for the next key or click to appear (#175).
- Reduced glass paints opaque fills and a short shadow, so it is really cheaper to draw: a 96% fill was still blended over the wallpaper on every frame, and the deep shadow was redrawn while scrolling. Thanks to @tian2992 (#174).

## [1.14.0] - 2026-11-17

### Added
- All servers: **Run a command…** runs one command on several servers, each in its own terminal tab, at most four at a time. A tab that asks something (a sudo password, `[Y/n]`) comes forward and you answer there; Davish never types a sudo password for a run. The confirm dialog lists every server and the exact command, and asks you to type the number of servers from three servers or for a command that looks destructive. Exit code per server, Stop all, Run again on the failed ones, and **Update packages** with each server's package manager and `sudo -n`. One line per server in the Command log.
- Command log: **Changes only** (on by default) hides reads and polling, and **Export script…** turns the ticked commands into a bash script that runs them again, each over its own `ssh -t` so sudo can ask. One block per server, Davish's sudo written as a plain `sudo`, content sent on stdin left as a comment (it is never recorded), failed commands only if you include them. Copy it or save it as an executable file.
- Files: the search box has three scopes. **This folder** filters as before; **Subfolders** finds names in every subfolder with `find`; **Contents** finds the lines that contain the text with `grep` (binary files, `.git` and `node_modules` are skipped). Press Enter to search the server: up to 500 results within 30 seconds, with the exact command that ran, a Stop button and a count of the places that could not be read. A content result opens in the Editor at its line; the folder button shows the file in its folder. Read only, never retried through sudo.
- Logs: **Search all** (or Enter in the filter) searches the whole history of a file, the journal or a container on the server, not just the 200 lines on screen, and returns the latest 500 matching lines. For a file, click a result to read the lines around it and page earlier or later without downloading the file (a log of several GB stays quick).
- Import servers from an Ansible inventory (INI): a new tab in the import dialog next to `~/.ssh/config`. It reads `ansible_host`, `ansible_user`, `ansible_port`, the key file and a `ProxyJump` in `ansible_ssh_common_args`, with group variables, child groups and ranges such as `web[01:03]`, and lets you pick a whole group at a time. The file is only read; passwords in it are never read, and hosts with templates, Vault values or a non-SSH connection are listed as skipped with the reason. A YAML inventory says it is not supported yet.

### Fixed
- Database: when the server has no sqlite3, the Find dialog offers the sqlite3 install command for the server's package manager with a Run in Terminal button (the dialog closes so Terminal is not hidden behind it), instead of only pointing to Packages. Each section of the Database app is now a card.
- macOS and Linux: text that should be selectable can be selected and copied again (file paths in the Find dialog, commands in confirm dialogs, Git diffs…). WebKit only reads `-webkit-user-select`, so these places could not be selected.
- Server menu: with many servers the list scrolls, so All servers, Add a server and Import stay within reach (#178).

## [1.13.10] - 2026-11-16

### Fixed
- Installing Docker from Setup also installs the compose plugin (a missing compose package does not stop the install). Nginx's HTTPS dialog checks for certbot first and offers its install command instead of a certbot line that would only fail. Setup says whether the tool is installed already, and no longer fails exactly when the tool is missing. Docker older than 23 reports "Docker is too old" for volumes instead of an unclear error.
- Windows: **Automatic** login now reads the default key files in `~\.ssh` (`id_ed25519`, `id_ecdsa`, `id_rsa`). The path, written with `\`, was never expanded to the home folder, so these files were never tried (#168).
- An SSH agent that does not answer (a pipe nobody serves, an agent waiting for its vault to unlock) no longer leaves the connection stuck on "Connecting…": after 5 seconds Davish moves on to the key files and writes why in the log (#168).
- A key path that points at a folder (such as `~/.ssh`) now says to choose the private key file in it, instead of "Incorrect function" on Windows (#168).

## [1.13.9] - 2026-11-14

### Fixed
- Windows: **Automatic** login uses the keys in the SSH agent of Windows (the “OpenSSH Authentication Agent” service, and the 1Password and Bitwarden agents that listen on the same pipe). Davish only looked for an agent through `SSH_AUTH_SOCK`, which Windows does not set, so only the key files in `~/.ssh` were tried (#164). When no key is found, the message says how to start the service.

## [1.13.8] - 2026-11-14

### Fixed
- Private keys encrypted with aes256-gcm, aes128-gcm, chacha20-poly1305, aes128/192-ctr or aes128/192-cbc (what `ssh-keygen -Z`, Termix and some other tools write) can be used: Davish only read aes256-ctr and aes256-cbc keys and failed with "unknown cipher" (#162). A saved passphrase that does not open the key now says so instead of "Could not read the private key file".

## [1.13.7] - 2026-11-14

### Added
- Git: each commit in the history shows the author's picture, the GitHub avatar of a noreply address or else the Gravatar of the email, over initials colored from the name (the initials stay when no picture loads or the machine is offline). To find a picture, the author's email (or its SHA-256 hash for Gravatar) is sent to GitHub's or Gravatar's image server. Scrolling near the end loads the next commits on its own, up to 1000; Load more only shows past that.
- Apps with a list and a detail (Git, Services, Docker…): drag the line between them to make the list wider or narrower, kept per app; a double click puts it back, arrow keys move it too.

### Changed
- Scroll bars are a thin faint pill, clearer while the pointer is over the area and clearest on the bar itself.

### Fixed
- Connecting: when a server cannot connect, a dialog now says why, with a report (Davish version, OS, auth method, error code, the technical cause, the connection's log lines) you can select or copy into a bug report. It never holds a password, key, host or user name, and your home folder shows as `~`. Clicking the red failure line on a server card opens it again. Before, some failures only said "Something went wrong" and lost their cause, even in the log (#162).
- A private key file saved by Notepad with a byte order mark could not be read; it can now.

## [1.13.6] - 2026-11-13

### Added
- Docker: containers as cards, one per compose project (or lone container), with a technology logo when Davish knows the image (n8n, WordPress, Postgres…), how many of its containers run, its image, published ports that open in the Browser, CPU and RAM; a red edge when one keeps restarting, and a click shows and controls its containers. A compose project's card takes the project's own favicon, or the framework named in `package.json`, `requirements.txt`, `composer.json`, `Gemfile`, `go.mod`… (Next, Nest, Express, Django, FastAPI, Laravel…), read from the project folder over SFTP; nothing runs on the server. A Cards/Table switch in the toolbar; the table by default, and Davish remembers your choice.
- All servers: **Map** on a connected server's card shows where requests go: each nginx site as a chain `domain → nginx :443 → :3000 → container (stack)`, static sites with their folder, and ports open to the outside without a proxy. Alerts come first: a database port open to the Internet, an upstream nobody listens on, a certificate expiring within 14 days, each with a button to the app that fixes it. Below, what runs and the disks. Read only. With four servers or more, a name filter and servers with alerts first.
- Monitor → Overview: the machine's details at the bottom of the page (the separate Info section and the Load · Uptime · Swap · GPU strip are gone: load sits under the CPU figure, swap in use and GPUs join the Hardware card, GPUs only when there are some), four cards, System (boot time, time zone, systemd, who is logged in), Hardware (cores, threads, sockets), Disks (NVMe/SSD/HDD/virtual drives with model, size, partitions and mounts; which mount holds Docker's data and whose volumes and bind mounts, through `docker info` and `docker inspect`, logged in the command log) and Network (LAN, VPN and Docker interfaces with addresses, gateway, DNS). Read only, through `lsblk`, `ip` and `/proc`.
- Launchpad: a **Recent** row at the top of All, with the five apps opened last (from the Dock, Spotlight or Launchpad), remembered across restarts.
- Settings → Dock: put the Dock on the bottom, left, right or top edge (windows and notifications keep clear; autohide comes back from that edge), and a transparent Dock with only the icons.

### Fixed
- Logs: text can be selected and copied. On macOS and Linux WebKit only reads `-webkit-user-select`, so every log line was locked. While a selection is in the view the log holds still (no scrolling, no new lines) so the selection does not slide, and goes on once it is cleared; `⌘C` copies the lines with their breaks. A new button copies the lines shown (after the filter).

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
