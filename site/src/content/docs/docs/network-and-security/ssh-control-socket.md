---
title: "Logins that only work through OpenSSH"
description: "For an agent that issues a certificate on demand (authentik), start an ssh master yourself and let Davish attach to it through its control socket."
sidebar:
  order: 7
---

Some logins cannot be done by Davish itself. An agent that issues a short-lived certificate on demand, such as the authentik agent, lists no keys until the client sends `session-bind@openssh.com`, and Davish's own SSH client cannot send it. OpenSSH can. For these servers Davish attaches to a connection that OpenSSH has already made: OpenSSH does the login, and Davish never sees your credentials.

This works on Linux and macOS only.

## Set it up

1. Start an ssh master for the server, from a terminal:

   ```sh
   ssh -M -S ~/.ssh/davish-myserver.sock -fN user@host.example.com
   ```

   If the agent is not your default one, add `-o IdentityAgent=/path/to/agent.sock`. Approve whatever the agent asks. `ssh` checks the host key the usual way, through `~/.ssh/known_hosts`.
2. In Davish, add or edit the server. Open **Advanced: use an ssh control socket** and enter the path of the socket from step 1: an absolute path, or one that starts with `~/`.
3. Save and connect. The Terminal, Files and the other apps work as on any server.

That field replaces the authentication of that server, and it cannot be combined with a jump server. Davish only attaches: it does not start or stop the master. If the master is gone, Davish says so, and you start it again.

Keep the path short. A Unix socket path is limited to about 104 bytes on macOS and 108 on Linux, and `ssh` fails with "ControlPath too long" beyond that.

## Keep the master running (Linux)

The master ends with a reboot, a logout or a network drop. A `systemd --user` service can start it at login and restart it. Save this as `~/.config/systemd/user/davish-master-myserver.service`, change the host, and add `-o IdentityAgent=...` if you need it:

```ini
[Unit]
Description=ssh master for Davish (myserver)

[Service]
ExecStartPre=-/usr/bin/rm -f %t/davish-myserver.sock
ExecStart=/usr/bin/ssh -M -S %t/davish-myserver.sock -N \
  -o ServerAliveInterval=30 -o ServerAliveCountMax=3 -o ExitOnForwardFailure=yes \
  user@host.example.com
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
```

Then:

```sh
systemctl --user daemon-reload
systemctl --user enable --now davish-master-myserver.service
ssh -S "$XDG_RUNTIME_DIR/davish-myserver.sock" -O check user@host.example.com
```

The last command should print `Master running`. In Davish, use the path `$XDG_RUNTIME_DIR/davish-myserver.sock` written out in full (`echo $XDG_RUNTIME_DIR` shows it, something like `/run/user/1000`). That folder is private to you.

Two details matter:

- `ExecStartPre` removes an old socket. If the master was killed, its socket stays behind, and `ssh -M` then prints "ControlSocket ... already exists, disabling multiplexing" and runs as a plain connection, without a master, so Davish cannot attach.
- The service has no terminal, so `ssh` cannot ask whether to trust a new host. Run `ssh user@host.example.com` by hand once first, so the host key is in `~/.ssh/known_hosts`, and make sure the agent can answer without you at a terminal (for authentik, that is its desktop prompt).

On macOS, run the same `ssh` command from whatever you use to start things at login.

## Security

Anyone who can open the socket gets a shell as you, without logging in. Davish refuses a socket that belongs to another user or that others can reach (`ssh` creates it with the right mode, or `chmod 600` it). A folder only you can enter, such as `$XDG_RUNTIME_DIR`, is a good place for it.
