package packages

import (
	"reflect"
	"testing"
)

func TestParseInstalledApt(t *testing.T) {
	out := "ii \tnginx\t1.24.0-2ubuntu7.19\t1347\tsmall, powerful, scalable web/proxy server\n" +
		"rc \told-pkg\t1.0\t10\tremoved, config left\n" +
		"ii \tlibssl3t64\t3.0.13-0ubuntu3.6\t7000\tSecure Sockets Layer toolkit\n"
	got := parseInstalled(Apt, []byte(out))
	want := []Package{
		{Name: "libssl3t64", Version: "3.0.13-0ubuntu3.6", Size: 7000 * 1024, Summary: "Secure Sockets Layer toolkit"},
		{Name: "nginx", Version: "1.24.0-2ubuntu7.19", Size: 1347 * 1024, Summary: "small, powerful, scalable web/proxy server"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestParseInstalledRpmAndApk(t *testing.T) {
	rpm := "openssl\t3.0.7-27.el9\t1792046\tUtilities from the general purpose cryptography library\ngpg-pubkey\tabc-def\t0\tgpg key\n"
	if got := parseInstalled(Dnf, []byte(rpm)); len(got) != 1 || got[0].Name != "openssl" || got[0].Size != 1792046 {
		t.Errorf("rpm = %+v", got)
	}
	apk := "musl-1.2.4-r2\nlibcrypto3-3.1.4-r5\nWARNING: junk\n"
	got := parseInstalled(Apk, []byte(apk))
	if len(got) != 2 || got[0] != (Package{Name: "libcrypto3", Version: "3.1.4-r5"}) || got[1].Name != "musl" {
		t.Errorf("apk = %+v", got)
	}
}

func TestParseUpdatesApt(t *testing.T) {
	out := "Listing...\n" +
		"openssl/noble-updates,noble-security 3.0.13-0ubuntu3.7 amd64 [upgradable from: 3.0.13-0ubuntu3.6]\n" +
		"nginx/noble-updates 1.24.0-2ubuntu7.20 amd64 [upgradable from: 1.24.0-2ubuntu7.19]\n"
	got := parseUpdates(Apt, []byte(out), nil)
	want := []Update{
		{Name: "nginx", Current: "1.24.0-2ubuntu7.19", Available: "1.24.0-2ubuntu7.20"},
		{Name: "openssl", Current: "3.0.13-0ubuntu3.6", Available: "3.0.13-0ubuntu3.7", Security: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v", got)
	}
}

func TestParseUpdatesDnfWithAdvisories(t *testing.T) {
	out := "\nopenssl.x86_64         1:3.0.7-28.el9     baseos\nopenssl-libs.x86_64    1:3.0.7-28.el9     baseos\n" +
		"kernel.x86_64          5.14.0-427.el9     baseos\nObsoleting Packages\n    grub2.noarch  1:2.06  baseos\n"
	ups := parseUpdates(Dnf, []byte(out), map[string]string{"openssl": "1:3.0.7-27.el9"})
	if len(ups) != 3 || ups[1].Name != "openssl" || ups[1].Current != "1:3.0.7-27.el9" {
		t.Fatalf("updates = %+v", ups)
	}
	adv := "ALSA-2024:1234 Important/Sec. openssl-libs-1:3.0.7-28.el9.x86_64\n"
	markSecurity(ups, parseAdvisories([]byte(adv)))
	for _, u := range ups {
		if u.Security != (u.Name == "openssl-libs") {
			t.Errorf("%s security = %v", u.Name, u.Security)
		}
	}
}

func TestParseUpdatesApk(t *testing.T) {
	got := parseUpdates(Apk, []byte("libcrypto3-3.1.4-r5 < 3.1.4-r6\nmusl-1.2.4-r2 = 1.2.4-r2\n"), nil)
	if len(got) != 1 || got[0] != (Update{Name: "libcrypto3", Current: "3.1.4-r5", Available: "3.1.4-r6"}) {
		t.Errorf("got %+v", got)
	}
}

func TestParseListsAge(t *testing.T) {
	if got := parseListsAge([]byte("1761600000\n1761596400\n")); got != 3600 {
		t.Errorf("age = %d", got)
	}
	if got := parseListsAge([]byte("1761600000\n")); got != -1 {
		t.Errorf("never fetched = %d", got)
	}
}

func TestPlanAction(t *testing.T) {
	tests := []struct {
		m, op string
		names []string
		want  string
	}{
		{Apt, OpUpgrade, []string{"nginx"}, "sudo apt-get install --only-upgrade nginx"},
		{Apt, OpUpgradeSecurity, []string{"openssl", "libssl3t64"}, "sudo apt-get install --only-upgrade openssl libssl3t64"},
		{Apt, OpUpgradeAll, nil, "sudo apt-get update && sudo apt-get upgrade"},
		{Apt, OpInstall, []string{"g++"}, "sudo apt-get install g++"},
		{Dnf, OpUpgradeSecurity, nil, "sudo dnf upgrade --security"},
		{Yum, OpRemove, []string{"httpd"}, "sudo yum remove httpd"},
		{Apk, OpUpgrade, []string{"musl"}, "sudo apk add -u musl"},
		{Apk, OpRefresh, nil, "sudo apk update"},
	}
	for _, tt := range tests {
		got, err := PlanAction(tt.m, tt.op, tt.names)
		if err != nil || got != tt.want {
			t.Errorf("%s %s %v = %q, %v; want %q", tt.m, tt.op, tt.names, got, err, tt.want)
		}
	}
	for _, bad := range []struct {
		m, op string
		names []string
	}{
		{Apt, OpInstall, nil},
		{Apt, OpUpgradeAll, []string{"x"}},
		{Apt, OpInstall, []string{"-y"}},
		{Apt, OpInstall, []string{"a;rm"}},
		{"pacman", OpInstall, []string{"x"}},
	} {
		if got, err := PlanAction(bad.m, bad.op, bad.names); err == nil {
			t.Errorf("%s %s %v planned %q", bad.m, bad.op, bad.names, got)
		}
	}
}
