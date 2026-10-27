# Local SSH server for integration tests only. Never expose it outside localhost.
# systemd runs as PID 1 so Services and Logs tests talk to a real systemctl
# and journald (docs/adr/0005-systemd-test-sshd.md).
FROM ubuntu:24.04

ENV container=docker

RUN apt-get update \
 && apt-get install -y --no-install-recommends systemd systemd-sysv dbus polkitd \
      openssh-server procps zip nginx \
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
# file to change. Test container only.
RUN chown -R tester:tester /etc/nginx

# Units for the Services tests: one that keeps running, one that always
# fails. tester may start, stop and restart davish-* units without sudo,
# and nothing else, so tests cover both success and "needs root".
COPY units/ /etc/systemd/system/
COPY polkit-tester.rules /etc/polkit-1/rules.d/50-davish-tester.rules
RUN systemctl enable davish-ok.service davish-crash.service

EXPOSE 22
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
