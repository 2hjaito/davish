---
title: "Editor"
description: "CodeMirror 6 with highlighting for nginx, YAML, JSON, shell, env and config files."
sidebar:
  order: 5
---

CodeMirror 6 with highlighting for nginx, YAML, JSON, shell, env and config files. Atomic saves that keep mode and owner, backups under `/etc`, and a warning if the file changed on the server. A file you cannot write opens read-only, with **Edit with sudo…**: saving then shows the exact command and writes the file as root, or use `sudoedit` in a terminal.

Settings → Editor sets the font size, wrapping of long lines, line numbers, marks for spaces and tabs (to spot a tab in a YAML file) and what the Tab key inserts (2 spaces, 4 spaces or a tab), next to the backup choice.
