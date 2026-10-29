# Contributing to Davish

Thanks for wanting to help. Bug reports, ideas and pull requests are welcome.

## Before you open a pull request

1. **Sign the CLA.** Davish is licensed under the GPL-3.0, and the maintainer may also offer it under other terms. The [Contributor License Agreement](CLA.md) lets your work be part of both. The CLA check on your first pull request explains how to sign; it takes one comment.
2. **Open an issue first** for anything bigger than a small fix, so we agree on the approach before you write it.
3. **Keep to the project rules:**
   - Nothing is installed on the server; everything goes over SSH and SFTP.
   - Every shell argument is quoted with `internal/shellx`; never build a command with string formatting.
   - Anything destructive (deleting, stopping, overwriting) shows the exact command in a confirmation dialog first.
   - No password, passphrase, key or token in logs, config files, tests or error messages.
   - The Wails bindings in `internal/app` stay thin; logic lives in the other `internal/*` packages.
   - New dependencies need a reason in the pull request.

## Working on the code

```bash
wails dev            # run the app with hot reload
make check           # go vet, golangci-lint, go test, svelte-check, eslint, vitest
make sshd-up         # the throwaway test server (Docker)
make test-integration
```

Never point tests or scripts at a real server; use the test sshd in `test/sshd/`.

Commits follow Conventional Commits (`feat(git): add the Git app`), in English. Text shown in the UI goes in `frontend/src/lib/i18n/vi.ts`.

## Security issues

Do not open a public issue for a vulnerability. Email the maintainer at the address on the GitHub profile instead.
