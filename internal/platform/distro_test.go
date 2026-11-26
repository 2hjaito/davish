package platform

import "testing"

func TestDistro(t *testing.T) {
	cases := []struct{ name, out, want string }{
		{"ubuntu", "ID=ubuntu\nID_LIKE=debian\n", "ubuntu"},
		{"debian quoted", "ID=\"debian\"\n", "debian"},
		{"rocky", "ID=\"rocky\"\nID_LIKE=\"rhel centos fedora\"\n", "rocky"},
		{"opensuse leap", "ID=\"opensuse-leap\"\nID_LIKE=\"suse opensuse\"\n", "opensuse"},
		{"raspberry pi os", "ID=raspbian\nID_LIKE=debian\n", "raspberrypi"},
		{"pop is like ubuntu", "ID=pop\nID_LIKE=\"ubuntu debian\"\n", "ubuntu"},
		{"manjaro is like arch", "ID=manjaro\nID_LIKE=arch\n", "arch"},
		{"amazon linux stays plain", "ID=\"amzn\"\nID_LIKE=\"fedora\"\n", DistroLinux},
		{"unknown", "ID=nixos\n", DistroLinux},
		{"upper case", "ID=UBUNTU\n", "ubuntu"},
		{"empty", "", DistroLinux},
		{"noise", "something\n=\nID\n", DistroLinux},
		{"id after like", "ID_LIKE=debian\nID=ubuntu\n", "ubuntu"},
	}
	for _, c := range cases {
		if got := Distro(c.out); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
