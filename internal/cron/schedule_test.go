package cron

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		raw                       string
		minute, hour, dom, month  []int
		dow                       []int
		domAny, dowAny, rebooting bool
	}{
		{raw: "*/15 * * * *", minute: []int{0, 15, 30, 45}},
		{raw: "0 3 * * *", minute: []int{0}, hour: []int{3}},
		{raw: "30 9 * * mon-fri", minute: []int{30}, hour: []int{9}, dow: []int{1, 2, 3, 4, 5}},
		{raw: "0 0 1,15 * *", minute: []int{0}, hour: []int{0}, dom: []int{1, 15}},
		{raw: "0 12 * jan,jul sun", minute: []int{0}, hour: []int{12}, month: []int{1, 7}, dow: []int{0}},
		{raw: "0 0 * * 7", minute: []int{0}, hour: []int{0}, dow: []int{0}},
		{raw: "5/20 * * * *", minute: []int{5, 25, 45}},
		{raw: "@daily", minute: []int{0}, hour: []int{0}},
		{raw: "@reboot", rebooting: true},
	}
	for _, tt := range tests {
		s, err := ParseSchedule(tt.raw)
		if err != nil {
			t.Errorf("%s: %v", tt.raw, err)
			continue
		}
		if tt.rebooting {
			if !s.Reboot {
				t.Errorf("%s: not @reboot", tt.raw)
			}
			continue
		}
		check := func(name string, got, want []int, all int, lo int) {
			if want == nil {
				if len(got) != all || got[0] != lo {
					t.Errorf("%s %s = %v, want every value", tt.raw, name, got)
				}
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s = %v, want %v", tt.raw, name, got, want)
			}
		}
		check("minute", s.Minute, tt.minute, 60, 0)
		check("hour", s.Hour, tt.hour, 24, 0)
		check("dom", s.Dom, tt.dom, 31, 1)
		check("month", s.Month, tt.month, 12, 1)
		check("dow", s.Dow, tt.dow, 7, 0)
	}
	for _, bad := range []string{"", "* * * *", "60 * * * *", "* 24 * * *", "* * 0 * *", "*/0 * * * *", "5-1 * * * *", "@sometimes", "* * * foo *"} {
		if _, err := ParseSchedule(bad); !errors.Is(err, ErrSchedule) {
			t.Errorf("ParseSchedule(%q) = %v, want ErrSchedule", bad, err)
		}
	}
}

func TestNext(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatal(err)
	}
	at := func(s string) time.Time {
		v, err := time.ParseInLocation("2006-01-02 15:04", s, loc)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// 2026-10-27 is a Tuesday.
	now := at("2026-10-27 14:07")
	tests := []struct{ raw, want string }{
		{"*/15 * * * *", "2026-10-27 14:15"},
		{"0 3 * * *", "2026-10-28 03:00"},
		{"7 14 * * *", "2026-10-28 14:07"}, // strictly after now
		{"30 9 * * mon-fri", "2026-10-28 09:30"},
		{"0 0 * * sat", "2026-10-31 00:00"},
		{"0 0 1 * *", "2026-11-01 00:00"},
		{"0 0 29 2 *", "2028-02-29 00:00"},
		// Both day fields restricted: the 1st or any Monday.
		{"0 8 1 * mon", "2026-11-01 08:00"},
		{"@weekly", "2026-11-01 00:00"},
	}
	for _, tt := range tests {
		s, err := ParseSchedule(tt.raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := s.Next(now); !got.Equal(at(tt.want)) {
			t.Errorf("%s: next = %s, want %s", tt.raw, got.Format("2006-01-02 15:04 Mon"), tt.want)
		}
	}
	never, _ := ParseSchedule("0 0 30 2 *")
	if got := never.Next(now); !got.IsZero() {
		t.Errorf("Feb 30 runs at %s", got)
	}
	reboot, _ := ParseSchedule("@reboot")
	if !reboot.Next(now).IsZero() {
		t.Error("@reboot has a next time")
	}
}
