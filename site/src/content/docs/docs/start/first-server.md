---
title: Connect your first server
description: Add a server with a key or a password, check its host key, or import the hosts you already have.
sidebar:
  order: 2
---

The first time you open Davish, it asks for a server. After that it connects to the server you used last as soon as it opens.

## Add a server

Click **Add server…** (from the welcome screen, the server menu in the menu bar, or **All servers**) and fill in the host, the port (22 by default) and the user. Then pick how to log in:

- **Automatic**, like the `ssh` command: the keys in your SSH agent, then the default keys in `~/.ssh` (`id_ed25519`, `id_ecdsa`, `id_rsa`).
- **Private key**: a key file you choose, with its passphrase if it has one.
- **Password**.

Passwords and passphrases are kept in your system's keychain, never in a file.

## Check the host key

The first time Davish connects to a server, it shows the server's host key fingerprint and asks you to accept it, like `ssh` does. If that key ever changes, Davish refuses to connect to protect you, and tells you why.

## Import the servers you already have

Already have hosts in `~/.ssh/config` or an Ansible inventory? Import them in one go, keys and `ProxyJump` included: see [ssh config and Ansible](/docs/network-and-security/ssh-config-and-ansible/). Servers behind a bastion connect through it, each hop with its own host key check.

## Many servers

Each server is a workspace with its own windows. Switch with `⌘1` … `⌘9` (`Ctrl` on Windows and Linux) or the server menu, and see them all in [All servers](/docs/on-the-server/all-servers/). You can drag files from one server to another in [Files](/docs/on-the-server/files/).

## When root is needed

Davish connects as your user. Actions that need root show the exact command first, then run through `sudo` with the password asked once per session, or go to a terminal with `sudo` typed in. See [Not root?](/docs/services-and-apps/not-root/).
