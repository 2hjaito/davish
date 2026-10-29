# Security

Davish holds the keys to your servers, so its design puts safety before convenience. This page explains what it stores, what it runs on your servers and how to report a vulnerability.

## Reporting a vulnerability

Please report privately through [GitHub Security Advisories](https://github.com/2hjaito/davish/security/advisories/new). Do not open a public issue. Include the version, your OS and steps to reproduce. You will get an answer within a few days; a fix ships in a release before the advisory is published.

Only the latest release is supported.

## Where secrets live

- **Passwords and passphrases** are stored only in the OS keychain (macOS Keychain, Windows Credential Manager, Secret Service on Linux). They are never written to files or logs and never sent to the UI.
- **Private keys** stay where they are. Davish stores the path to your key (usually in `~/.ssh/`), never a copy. An SSH agent is used when available.
- **`sudo` passwords** are never stored or typed by Davish. When root is needed, it opens a terminal with the command typed in and you press Enter.
- **The lock password** is hashed with argon2id; only the hash is kept, in the keychain.
- Removing a server also removes its secrets from the keychain.

## What Davish runs on your server

- Nothing is installed or left behind. Davish needs only `sshd` and SFTP and runs everything from your machine over SSH.
- No new port is opened on the server.
- Every argument in a command is shell-quoted; user input is never pasted into a command line.
- Destructive actions (removing files, stopping services, removing containers, replacing a crontab…) show the exact command and wait for your confirmation.
- The command log records everything Davish ran on your behalf.
- Host keys are checked on every connection. A changed host key refuses the connection; there is no "ignore" button.

## Network and telemetry

- No telemetry, analytics or crash reporting.
- Besides your servers, Davish talks to one place: the GitHub Releases API, to check for updates. Updates are verified against a signature before they are installed.

## Out of scope

The app lock protects against someone using your unlocked computer. It does not encrypt data at rest; use full-disk encryption for that.
