# Local SSH server for integration tests only. Never expose it outside localhost.
FROM ubuntu:24.04

RUN apt-get update \
 && apt-get install -y --no-install-recommends openssh-server procps zip \
 && rm -rf /var/lib/apt/lists/* \
 && mkdir -p /run/sshd \
 && useradd -m -s /bin/bash tester \
 && rm -f /etc/ssh/ssh_host_*

# Fixed host key so tests can pin it instead of skipping verification.
COPY keys/host_ed25519 /etc/ssh/ssh_host_ed25519_key
COPY keys/client_ed25519.pub /home/tester/.ssh/authorized_keys
COPY sshd_config /etc/ssh/sshd_config.d/davish-test.conf

RUN chmod 600 /etc/ssh/ssh_host_ed25519_key \
 && chown -R tester:tester /home/tester/.ssh \
 && chmod 700 /home/tester/.ssh \
 && chmod 600 /home/tester/.ssh/authorized_keys

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D", "-e"]
