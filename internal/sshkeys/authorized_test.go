package sshkeys

import (
	"strings"
	"testing"
)

const edKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEp6HY7f4AmOW4XWXa+YEbRsVYDhpKwnCo6E7fxKBGYH"

func TestParseAuthorized(t *testing.T) {
	data := "# managed by hand\n\n" +
		edKey + " dang@laptop\n" +
		`from="10.0.0.0/8",no-pty ` + edKey + " ci\r\n" +
		"not a key\n"
	keys := parseAuthorized([]byte(data))
	if len(keys) != 2 {
		t.Fatalf("keys = %+v", keys)
	}
	if k := keys[0]; k.Type != "ssh-ed25519" || k.Comment != "dang@laptop" || !strings.HasPrefix(k.Fingerprint, "SHA256:") || k.Text != edKey+" dang@laptop" {
		t.Errorf("first = %+v", k)
	}
	if k := keys[1]; len(k.Options) != 2 || k.Options[1] != "no-pty" || k.Comment != "ci" || strings.HasSuffix(k.Text, "\r") {
		t.Errorf("second = %+v", k)
	}
	if keys[0].Fingerprint != keys[1].Fingerprint {
		t.Error("the same key with options got another fingerprint")
	}
}

func TestCommands(t *testing.T) {
	file := "/home/dang/.ssh/authorized_keys"
	add := addCommand(file, edKey+" dang's key")
	want := "mkdir -p -m 700 -- /home/dang/.ssh && touch -- " + file + " && chmod 600 -- " + file +
		" && { grep -qxF -- '" + edKey + " dang'\\''s key' " + file +
		" || printf '%s\\n' '" + edKey + " dang'\\''s key' >> " + file + "; }"
	if add != want {
		t.Errorf("add:\n got %s\nwant %s", add, want)
	}

	rm := removeCommand(file, file+".bak-20261028T120000Z", []string{edKey + " a", edKey + " b"})
	want = "cp -p -- " + file + " " + file + ".bak-20261028T120000Z" +
		" && { grep -vxF -e '" + edKey + " a' -e '" + edKey + " b' -- " + file + " > " + file + ".davish-tmp || [ $? -eq 1 ]; }" +
		" && chmod 600 -- " + file + ".davish-tmp && mv -- " + file + ".davish-tmp " + file
	if rm != want {
		t.Errorf("remove:\n got %s\nwant %s", rm, want)
	}
}
