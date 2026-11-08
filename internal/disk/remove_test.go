package disk

import (
	"context"
	"testing"
)

func TestProtected(t *testing.T) {
	for _, p := range []string{
		"/", "/usr", "/usr/local/bin", "/etc/nginx", "/var", "/var/log", "/var/lib",
		"/var/lib/docker/overlay2/abc", "/var/lib/mysql/ibdata1", "/home", "/home/deploy",
		"/root", "/boot/vmlinuz", "/proc/1", "/tmp",
	} {
		if !Protected(p) {
			t.Errorf("%s is not protected", p)
		}
	}
	for _, p := range []string{
		"/var/log/nginx/access.log.1", "/home/deploy/backups", "/root/old.tar.gz",
		"/opt/app/node_modules", "/srv/site/uploads", "/tmp/build", "/var/cache/apt/archives",
		"/usrlocal", "/var/lib/dockerish",
	} {
		if Protected(p) {
			t.Errorf("%s is protected", p)
		}
	}
}

func TestPlanTruncate(t *testing.T) {
	got, err := PlanTruncate("/var/log/app it's.log")
	if err != nil || got.Command != `truncate -s 0 -- '/var/log/app it'\''s.log'` ||
		got.Sudo != `sudo truncate -s 0 -- '/var/log/app it'\''s.log'` {
		t.Errorf("plan = %+v, %v", got, err)
	}
	if _, err := PlanTruncate("/var/lib/docker/containers/abc/abc-json.log"); err != nil {
		t.Errorf("container log refused: %v", err)
	}
	for _, p := range []string{"/", "/etc/passwd", "/var/lib/mysql/ibdata1", "/var/lib/docker/overlay2/x", "relative.log", "/var/log/../etc/passwd"} {
		if got, err := PlanTruncate(p); err == nil {
			t.Errorf("%s planned %+v", p, got)
		}
	}
}

func TestRemoveRefusesProtectedFirst(t *testing.T) {
	// A nil remover would panic if a protected path reached it.
	s := New(nil, nil)
	if _, err := s.PlanRemove(context.Background(), "srv", "/usr/lib"); err == nil {
		t.Error("planned removing /usr/lib")
	}
	if err := s.Remove(context.Background(), "srv", "/home/deploy", "rm -r -- /home/deploy"); err == nil {
		t.Error("removed a home folder")
	}
	if err := s.Truncate(context.Background(), "srv", "/var/log/a.log", "truncate -s 0 -- /var/log/b.log"); err == nil {
		t.Error("ran a command other than the confirmed one")
	}
}
