#!/bin/sh
# Creates the containers the Docker tests expect, from the local busybox.
# Test container only.
set -eu
if ! docker image inspect davish/busybox >/dev/null 2>&1; then
  tar -C /opt/davish/busybox -c . | docker import -c 'CMD ["/bin/sh"]' - davish/busybox
fi
docker rm -f davish-web davish-crashloop >/dev/null 2>&1 || true
# Keeps running and logs a line a second.
docker run -d --name davish-web --restart unless-stopped davish/busybox \
  sh -c 'i=0; while :; do i=$((i+1)); echo "request $i served"; sleep 1; done'
# Dies at once and is restarted for ever, with an error naming a file.
docker run -d --name davish-crashloop --restart always davish/busybox \
  sh -c 'echo "Error: cannot open /etc/nginx/nginx.conf: no such file" >&2; exit 1'
