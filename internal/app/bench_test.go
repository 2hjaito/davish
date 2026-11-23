package app

import (
	"strings"
	"testing"
)

func TestBenchLine(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "scenario=idle frames=180", "davish-bench scenario=idle frames=180"},
		{"line breaks", "a=1\ndavish-bench ready_ms=1\r\nb=2", "davish-bench a=1 davish-bench ready_ms=1  b=2"},
		{"empty", "", "davish-bench "},
	}
	for _, c := range cases {
		if got := benchLine(c.in); got != c.want {
			t.Errorf("%s: benchLine(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
		if strings.ContainsAny(benchLine(c.in), "\r\n") {
			t.Errorf("%s: result has a line break", c.name)
		}
	}
}

func TestBenchLineIsBounded(t *testing.T) {
	got := benchLine(strings.Repeat("x", 5000))
	if want := len("davish-bench ") + maxBenchLine; len(got) != want {
		t.Errorf("len = %d, want %d", len(got), want)
	}
}

func TestBenchConfigOff(t *testing.T) {
	t.Setenv("DAVISH_BENCH", "")
	if got := (&App{}).BenchConfig(); got != (BenchConfig{}) {
		t.Errorf("BenchConfig() = %+v outside a bench", got)
	}
	t.Setenv("DAVISH_BENCH", "1") // the startup bench: no render scenarios
	if got := (&App{}).BenchConfig(); got.Render {
		t.Errorf("Render = true for the startup bench")
	}
}

func TestBenchConfigRender(t *testing.T) {
	t.Setenv("DAVISH_BENCH", "render")
	t.Setenv("DAVISH_BENCH_PRESET", "light")
	got := (&App{}).BenchConfig()
	if !got.Render || got.Preset != "light" {
		t.Errorf("BenchConfig() = %+v", got)
	}
}
