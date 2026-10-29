#!/usr/bin/env bash
# Sets the GitHub repo About section: description, homepage and topics.
# Needs the gh CLI, logged in with admin rights on the repo.
set -euo pipefail

REPO="${REPO:-2hjaito/davish}"

gh repo edit "$REPO" \
  --description "A desktop for your Linux servers. No web panel, no login, no agent. Open it and you're in, over SSH." \
  --homepage "https://davish.dangth.dev" \
  --add-topic ssh,sftp,vps,server-management,linux,devops,sysadmin,desktop-app \
  --add-topic wails,golang,svelte,docker,nginx,terminal,self-hosted
