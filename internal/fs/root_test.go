package fs

import (
	"strings"
	"testing"
)

func TestRootRemoveRefused(t *testing.T) {
	refused := []string{
		"/", "/etc", "/etc/", "/usr", "/usr/local/bin/x", "/bin/sh", "/boot/vmlinuz", "/proc/1",
		"/sys/kernel", "/dev/sda", "/run/sshd", "/lib/x86_64-linux-gnu/libc.so.6",
		"/var", "/var/log", "/var/lib", "/var/lib/dpkg/status", "/var/lib/rpm/x", "/var/www",
		"/home", "/home/alice", "/root", "/tmp", "/opt", "/srv",
		"/etc/passwd", "/etc/shadow", "/etc/group", "/etc/gshadow", "/etc/fstab",
		"/etc/sudoers", "/etc/sudoers.d/admins", "/etc/ssh", "/etc/ssh/sshd_config", "/etc/pam.d/sshd",
		"/etc/../etc/passwd", "/home/alice/../bob",
	}
	for _, p := range refused {
		if !RootRemoveRefused(p) {
			t.Errorf("%s is allowed", p)
		}
	}
	allowed := []string{
		"/etc/nginx/sites-enabled/shop", "/etc/cron.d/backup", "/var/log/nginx/access.log.1",
		"/var/www/old-site", "/home/alice/.cache", "/root/old.tar.gz", "/opt/app/releases/1",
		"/srv/data/tmp", "/tmp/x", "/etc/passwd-", "/etc/sshd_banner",
	}
	for _, p := range allowed {
		if RootRemoveRefused(p) {
			t.Errorf("%s is refused", p)
		}
	}
}

func TestRootWriteRefused(t *testing.T) {
	for _, p := range []string{"/proc/sys/vm/swappiness", "/sys/x", "/usr/bin/sudo", "/bin/sh", "/boot/grub/grub.cfg"} {
		if !RootWriteRefused(p) {
			t.Errorf("%s may be written", p)
		}
	}
	for _, p := range []string{"/etc/nginx/nginx.conf", "/etc/ssh/sshd_config", "/etc/hosts", "/root/.bashrc"} {
		if RootWriteRefused(p) {
			t.Errorf("%s may not be written", p)
		}
	}
}

func TestRootSaveCommand(t *testing.T) {
	got := rootSaveCommand("/etc/it's here.conf", "/etc/it's here.conf.bak-20261111T120000Z", 0, 4, 0o2640)
	want := `set -e; if test -e '/etc/it'\''s here.conf.bak-20261111T120000Z'; then exit 73; fi; ` +
		`cp -p -- '/etc/it'\''s here.conf' '/etc/it'\''s here.conf.bak-20261111T120000Z'; ` +
		`t=$(mktemp '/etc/.it'\''s here.conf.davish-XXXXXX'); trap 'rm -f -- "$t"' EXIT; ` +
		`cat > "$t"; chown 0:4 -- "$t"; chmod 2640 -- "$t"; mv -f -- "$t" '/etc/it'\''s here.conf'`
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	// File type bits never reach chmod.
	if c := rootSaveCommand("/etc/x", "", 0, 0, 0o100644); !strings.Contains(c, "chmod 644 ") || strings.Contains(c, "cp -p") {
		t.Errorf("plain file: %s", c)
	}
}
