package shellx

import (
	"os/exec"
	"testing"
)

var quoteCases = []struct {
	name, in, want string
}{
	{"empty", "", "''"},
	{"plain", "nginx.service", "nginx.service"},
	{"path", "/etc/nginx/nginx.conf", "/etc/nginx/nginx.conf"},
	{"space", "my file.txt", "'my file.txt'"},
	{"single quote", "it's", `'it'\''s'`},
	{"double quote", `say "hi"`, `'say "hi"'`},
	{"command substitution", "$(rm -rf /)", "'$(rm -rf /)'"},
	{"backticks", "`id`", "'`id`'"},
	{"newline", "a\nb", "'a\nb'"},
	{"semicolon", "a; reboot", "'a; reboot'"},
	{"glob", "*.log", "'*.log'"},
	{"leading dash", "-rf", "-rf"},
	{"unicode", "tệp.txt", "'tệp.txt'"},
}

func TestQuote(t *testing.T) {
	for _, tt := range quoteCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Quote(tt.in); got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestQuoteRoundTrip checks that a real shell sees exactly the original
// string as one argument.
func TestQuoteRoundTrip(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	for _, tt := range quoteCases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command(sh, "-c", "printf %s "+Quote(tt.in)).Output() //nolint:gosec // input is quoted by the code under test
			if err != nil {
				t.Fatal(err)
			}
			if string(out) != tt.in {
				t.Errorf("shell saw %q, want %q", out, tt.in)
			}
		})
	}
}

func TestCmdString(t *testing.T) {
	got := Cmd("systemctl", "restart", "my app.service").String()
	want := "systemctl restart 'my app.service'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestArgsIsCopy(t *testing.T) {
	c := Cmd("ls", "/tmp")
	a := c.Args()
	a[1] = "/etc"
	if c.Args()[1] != "/tmp" {
		t.Error("Args must return a copy")
	}
}

func TestInDir(t *testing.T) {
	got := InDir("/srv/my site", Cmd("tar", "-cf", "a b.tar", "--", "a b"))
	want := "cd -- '/srv/my site' && tar -cf 'a b.tar' -- 'a b'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
