# Local SSH server for integration tests only. Never expose it outside localhost.
# systemd runs as PID 1 so Services and Logs tests talk to a real systemctl
# and journald (docs/adr/0005-systemd-test-sshd.md).
# busybox's whole root to build the test containers from, with no registry
# pull when tests run (docs/adr/0006-docker-in-test-sshd.md).
FROM busybox:1.37 AS busybox

FROM ubuntu:24.04

ENV container=docker

RUN apt-get update \
 && apt-get install -y --no-install-recommends systemd systemd-sysv dbus polkitd \
      openssh-server procps zip nginx docker.io cron curl git \
 && rm -rf /var/lib/apt/lists/* \
 && useradd -m -s /bin/bash -G systemd-journal tester \
 && rm -f /etc/ssh/ssh_host_*

# Nothing a container needs: consoles, device manager, kernel mounts.
RUN systemctl mask getty.target console-getty.service systemd-udevd.service \
      systemd-udev-trigger.service systemd-remount-fs.service sys-kernel-debug.mount \
      sys-kernel-tracing.mount systemd-modules-load.service \
 && systemctl disable ssh.socket \
 && systemctl enable ssh.service

# Fixed host key so tests can pin it instead of skipping verification.
COPY keys/host_ed25519 /etc/ssh/ssh_host_ed25519_key
COPY keys/client_ed25519.pub /home/tester/.ssh/authorized_keys
COPY sshd_config /etc/ssh/sshd_config.d/davish-test.conf

RUN chmod 600 /etc/ssh/ssh_host_ed25519_key \
 && chown -R tester:tester /home/tester/.ssh \
 && chmod 700 /home/tester/.ssh \
 && chmod 600 /home/tester/.ssh/authorized_keys

# tester owns nginx's config, so Editor and Nginx tests have a real /etc
# file to change, and plays an admin who may test and reload nginx: it can
# write the pid file nginx -t opens and the logs, and polkit lets it reload
# nginx.service. Test container only.
RUN chown -R tester:tester /etc/nginx /var/log/nginx
COPY sites/ /etc/nginx/sites-available/
RUN mkdir -p /srv/davish-site && echo 'davish site' > /srv/davish-site/davish.html \
 && ln -s /etc/nginx/sites-available/davish-site /etc/nginx/sites-enabled/davish-site \
 && chown -R tester:tester /etc/nginx

# Docker runs inside this container, never the host's. tester is in the
# docker group, as on most servers that use Docker.
COPY --from=busybox / /opt/davish/busybox/
COPY containers.sh /opt/davish/containers.sh
RUN usermod -aG docker tester

# Units for the Services tests: one that keeps running, one that always
# fails. tester may start, stop and restart davish-* units without sudo,
# and nothing else, so tests cover both success and "needs root".
COPY units/ /etc/systemd/system/
COPY polkit-tester.rules /etc/polkit-1/rules.d/50-davish-tester.rules
RUN systemctl enable davish-ok.service davish-crash.service davish-containers.service docker.service cron.service

EXPOSE 22
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
