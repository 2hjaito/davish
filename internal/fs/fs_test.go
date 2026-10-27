package fs

import (
	"errors"
	"os"
	"testing"

	"davish/internal/apperr"
)

func TestLsMode(t *testing.T) {
	tests := []struct {
		mode uint32
		want string
	}{
		{sIFREG | 0o644, "-rw-r--r--"},
		{sIFDIR | 0o755, "drwxr-xr-x"},
		{sIFLNK | 0o777, "lrwxrwxrwx"},
		{sIFREG | sISUID | 0o4755, "-rwsr-xr-x"},
		{sIFREG | sISUID | 0o644, "-rwSr--r--"},
		{sIFDIR | sISGID | 0o2775, "drwxrwsr-x"},
		{sIFDIR | sISVTX | 0o1777, "drwxrwxrwt"},
		{sIFDIR | sISVTX | 0o1770, "drwxrwx--T"},
		{sIFCHR | 0o620, "crw--w----"},
		{sIFBLK | 0o660, "brw-rw----"},
		{sIFIFO | 0o600, "prw-------"},
		{sIFSOCK | 0o777, "srwxrwxrwx"},
		{0o600, "?rw-------"},
	}
	for _, tt := range tests {
		if got := lsMode(tt.mode); got != tt.want {
			t.Errorf("lsMode(%o) = %q; want %q", tt.mode, got, tt.want)
		}
	}
}

func TestKindOf(t *testing.T) {
	tests := map[uint32]string{
		sIFREG | 0o644: KindFile,
		sIFDIR | 0o755: KindDir,
		sIFLNK | 0o777: KindLink,
		sIFSOCK:        KindOther,
		sIFCHR:         KindOther,
	}
	for mode, want := range tests {
		if got := kindOf(mode); got != want {
			t.Errorf("kindOf(%o) = %q; want %q", mode, got, want)
		}
	}
}

func TestCheckPath(t *testing.T) {
	tests := []struct {
		p  string
		ok bool
	}{
		{"/", true},
		{"/etc/nginx/nginx.conf", true},
		{"/home/user/my file's (1).txt", true},
		{"", false},
		{"etc/passwd", false},
		{"/etc/", false},
		{"/etc/../root", false},
		{"/etc//passwd", false},
		{"/tmp/./x", false},
		{"/tmp/a\x00b", false},
	}
	for _, tt := range tests {
		err := CheckPath(tt.p)
		if (err == nil) != tt.ok {
			t.Errorf("CheckPath(%q) = %v; want ok=%v", tt.p, err, tt.ok)
		}
	}
}

func TestRemoveCommand(t *testing.T) {
	tests := []struct {
		kind     string
		nonEmpty bool
		p        string
		want     string
	}{
		{KindFile, false, "/tmp/a.txt", "rm -- /tmp/a.txt"},
		{KindLink, false, "/tmp/link", "rm -- /tmp/link"},
		{KindOther, false, "/tmp/sock", "rm -- /tmp/sock"},
		{KindDir, false, "/tmp/empty", "rmdir -- /tmp/empty"},
		{KindDir, true, "/tmp/full", "rm -r -- /tmp/full"},
		{KindFile, false, "/tmp/-rf $(reboot)'", `rm -- '/tmp/-rf $(reboot)'\'''`},
	}
	for _, tt := range tests {
		if got := removeCommand(tt.kind, tt.nonEmpty, tt.p).String(); got != tt.want {
			t.Errorf("removeCommand(%s, %v, %q) = %q; want %q", tt.kind, tt.nonEmpty, tt.p, got, tt.want)
		}
	}
}

func TestTmpName(t *testing.T) {
	if got := tmpName("/etc/nginx/nginx.conf"); got != "/etc/nginx/.nginx.conf.davish-tmp" {
		t.Errorf("tmpName = %q", got)
	}
}

func TestFileMode(t *testing.T) {
	if got := fileMode(sIFREG | sISUID | sISGID | sISVTX | 0o755); got != os.ModeSetuid|os.ModeSetgid|os.ModeSticky|0o755 {
		t.Errorf("fileMode = %v", got)
	}
}

func TestMapErr(t *testing.T) {
	tests := []struct {
		err  error
		code apperr.Code
	}{
		{os.ErrNotExist, apperr.NotFound},
		{os.ErrPermission, apperr.FSPermissionDenied},
		{apperr.New(apperr.FSExists, "x"), apperr.FSExists},
		{errors.New("boom"), apperr.Internal},
	}
	for _, tt := range tests {
		var ae *apperr.Error
		if err := mapErr("op", tt.err); !errors.As(err, &ae) || ae.Code != tt.code {
			t.Errorf("mapErr(%v) = %v; want %s", tt.err, err, tt.code)
		}
	}
}
