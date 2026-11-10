# Local SSH server for integration tests only. Never expose it outside localhost.
# systemd runs as PID 1 so Services and Logs tests talk to a real systemctl
# and journald (docs/adr/0005-systemd-test-sshd.md).
# busybox's whole root to build the test containers from, with no registry
# pull when tests run (docs/adr/0006-docker-in-test-sshd.md).
FROM busybox:1.37 AS busybox

FROM ubuntu:24.04

ENV container=docker

# Ubuntu's mirrors drop connections and serve 404s mid-sync now and then;
# apt tries each download again, with growing delays, instead of failing
# the build and the CI run with it.
RUN printf 'Acquire::Retries "8";\nAcquire::Retries::Delay "true";\nAcquire::http::Timeout "30";\n' \
      > /etc/apt/apt.conf.d/80-davish-retries

# A closer mirror, when given: CI passes Azure's, next to GitHub's runners,
# where archive.ubuntu.com can take a minute per index.
ARG UBUNTU_MIRROR=
RUN if [ -n "$UBUNTU_MIRROR" ]; then \
      sed -i "s|http://archive.ubuntu.com/ubuntu|$UBUNTU_MIRROR|g; s|http://security.ubuntu.com/ubuntu|$UBUNTU_MIRROR|g" \
        /etc/apt/sources.list.d/ubuntu.sources; \
    fi

# Temporary (2026-11-10): Ubuntu lists perl 5.38.2-3.2ubuntu0.7 in
# noble-security but its .debs 404 on every mirror, so installing git, which
# needs perl, failed every CI run. Keep perl at the version the base image
# ships, still in noble-updates. Remove once the 0.7 files are served.
RUN v=$(dpkg-query -W -f '${Version}' perl-base) \
 && printf 'Package: perl perl-base perl-modules-5.38 libperl5.38t64\nPin: version %s\nPin-Priority: 1001\n' "$v" \
      > /etc/apt/preferences.d/80-davish-perl

RUN apt-get update \
 && apt-get install -y --no-install-recommends systemd systemd-sysv dbus polkitd \
      openssh-server procps zip nginx docker.io cron curl git sudo \
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

# MongoDB on the host, as most servers run it: systemd, /etc/mongod.conf,
# authorization on, so Database tests cover the login (ADR 0017). The
# admin password is a throwaway for this container only.
RUN apt-get update \
 && apt-get install -y --no-install-recommends gnupg ca-certificates sqlite3 \
 && curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc \
      | gpg --dearmor -o /usr/share/keyrings/mongodb-server-8.0.gpg \
 && echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/ubuntu noble/mongodb-org/8.0 multiverse" \
      > /etc/apt/sources.list.d/mongodb-org-8.0.list \
 && apt-get update \
 && apt-get install -y --no-install-recommends mongodb-org-server mongodb-mongosh mongodb-database-tools \
 && rm -rf /var/lib/apt/lists/*
RUN mongod --fork --dbpath /var/lib/mongodb --logpath /tmp/mongod-setup.log --bind_ip 127.0.0.1 \
 && mongosh --quiet --eval 'db.getSiblingDB("admin").createUser({user: "davish-admin", pwd: "davish-test-only", roles: ["root"]})' \
 && mongosh --quiet -u davish-admin -p davish-test-only --authenticationDatabase admin \
      --eval 'db.getSiblingDB("shop").orders.insertMany([{n: 1}, {n: 2}, {n: 3}])' \
 && mongod --shutdown --dbpath /var/lib/mongodb \
 && rm -f /tmp/mongod-setup.log \
 && chown -R mongodb:mongodb /var/lib/mongodb \
 && printf 'security:\n  authorization: enabled\n' >> /etc/mongod.conf \
 && systemctl enable mongod.service

# SQLite files for Database's Find: one tester can read, one only root
# can, and a .db file that is not SQLite at all.
RUN mkdir -p /srv/davish-app /var/lib/davish-private \
 && sqlite3 /srv/davish-app/app.db \
      "CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT); INSERT INTO users(name) VALUES ('a'), ('b');" \
 && sqlite3 /var/lib/davish-private/private.sqlite "CREATE TABLE t(x); INSERT INTO t VALUES (1);" \
 && echo 'not a database' > /srv/davish-app/notes.db \
 && chown -R tester:tester /srv/davish-app \
 && chmod 700 /var/lib/davish-private

# sudoer is an admin as most servers have one: in the sudo group, and sudo
# asks for a password (ADR 0018). tester stays without sudo. The password
# is a throwaway for this container only.
RUN useradd -m -s /bin/bash -G sudo,systemd-journal sudoer \
 && echo 'sudoer:davish-sudo-test-only' | chpasswd \
 && mkdir -p /home/sudoer/.ssh \
 && cp /home/tester/.ssh/authorized_keys /home/sudoer/.ssh/authorized_keys \
 && chown -R sudoer:sudoer /home/sudoer/.ssh \
 && chmod 700 /home/sudoer/.ssh \
 && chmod 600 /home/sudoer/.ssh/authorized_keys

EXPOSE 22
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
