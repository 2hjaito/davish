package git

import (
	"errors"
	"strings"
	"testing"

	"davish/internal/apperr"
)

func TestFindLineQuotesFolders(t *testing.T) {
	line := findLine([]string{"/srv/my site", "/x/it's", "/x/$(reboot)", "/x/`id`"})
	for _, want := range []string{
		`find "$HOME" /var/www /srv /opt '/srv/my site' '/x/it'\''s' '/x/$(reboot)' '/x/` + "`id`" + `' -mindepth 1`,
		`-maxdepth 4`,
		`| head -n 500; true`,
	} {
		if !strings.Contains(line, want) {
			t.Errorf("line %q lacks %q", line, want)
		}
	}
}

func TestCheckExtra(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
		ok   bool
	}{
		{"none", nil, []string{}, true},
		{"cleaned", []string{"/home/deploy/apps/", "/srv//x/../y"}, []string{"/home/deploy/apps", "/srv/y"}, true},
		{"relative", []string{"apps"}, nil, false},
		{"option", []string{"-delete"}, nil, false},
		{"newline", []string{"/srv/a\n/b"}, nil, false},
		{"too many", make([]string, maxExtra+1), nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkExtra(tt.in)
			if !tt.ok {
				var ae *apperr.Error
				if !errors.As(err, &ae) || ae.Code != apperr.InvalidInput {
					t.Fatalf("err = %v, want INVALID_INPUT", err)
				}
				return
			}
			if err != nil || strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestParseFind(t *testing.T) {
	out := strings.Join([]string{
		"/var/www/shop/.git",
		"/home/deploy/api/.git",
		"/srv/home/deploy/api/.git",
		"/home/deploy/api/.git",
		"/opt/tool/.git\r",
		"find: '/root': Permission denied",
		"/.git",
		"",
	}, "\n")
	got := parseFind([]byte(out))
	want := []Repo{
		{"/home/deploy/api", "api"},
		{"/opt/tool", "tool"},
		{"/srv/home/deploy/api", "api"},
		{"/var/www/shop", "shop"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("repo %d = %v, want %v", i, got[i], want[i])
		}
	}
}
