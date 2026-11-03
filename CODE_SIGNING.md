# Code signing policy

Free code signing for the Windows builds is provided by [SignPath.io](https://about.signpath.io), certificate by [SignPath Foundation](https://signpath.org).

## What is signed

- `Davish-<version>-windows-amd64-setup.exe`, the Windows installer.
- `Davish.exe` inside `Davish-<version>-windows-amd64.zip`, the portable build.

Only files built by the [release workflow](.github/workflows/release.yml) from a tag in this repository are submitted for signing. Nothing built on a developer's machine is signed. Every signed file comes from source code in this repository and third-party open source components listed in `go.mod` and `frontend/package-lock.json`.

The macOS and Linux builds are not covered by this policy.

Independently of code signing, the checksum list of each release (`SHA256SUMS.txt`) is signed with a key that the in-app updater trusts, so updates installed from inside Davish are verified on every platform.

## Team roles

| Role | Members |
|---|---|
| Committers and reviewers | [Tran Huu Dang](https://github.com/2hjaito) |
| Approvers | [Tran Huu Dang](https://github.com/2hjaito) |

Changes from other contributors reach `main` only through a pull request reviewed by a committer. Each signing request is approved by an approver. Members of these roles use multi-factor authentication on GitHub and SignPath.

## Privacy

Davish connects to the servers you add, over SSH. Everything it does on a server, including the web pages shown in its built-in browser, goes through that SSH connection. It does not collect usage data, telemetry or crash reports.

The one other connection is the **update check**: a request to `api.github.com` at startup and every 6 hours to look for a new version, and the download from GitHub Releases when you choose to update. No user or device identifier is sent. Turn it off with Settings → Updates → Check for new versions; a manual check is still available in the menu.

This program will not transfer any other information to other networked systems unless specifically requested by the user or the person installing or operating it.

## Reporting

Report a signed file that looks wrong, or a breach of this policy, as described in [SECURITY.md](SECURITY.md).
