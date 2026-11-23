package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// BenchConfig is what the render bench (scripts/bench.sh --render) asks of
// this run. Outside a bench both fields are zero and nothing else here does
// anything: DAVISH_BENCH is set by the script, never by the app.
type BenchConfig struct {
	// Render is true when the frontend should run the render scenarios.
	Render bool `json:"render"`
	// Preset is the performance preset to measure: beautiful, balanced or light.
	Preset string `json:"preset"`
	// Widgets is how many widgets to put on the desktop before measuring.
	Widgets int `json:"widgets"`
}

// BenchConfig reads the bench variables the script set for this process.
func (a *App) BenchConfig() BenchConfig {
	if os.Getenv("DAVISH_BENCH") != "render" {
		return BenchConfig{}
	}
	return BenchConfig{
		Render:  true,
		Preset:  os.Getenv("DAVISH_BENCH_PRESET"),
		Widgets: benchWidgets(os.Getenv("DAVISH_BENCH_WIDGETS")),
	}
}

// maxBenchWidgets is as many as one desktop holds.
const maxBenchWidgets = 12

// benchWidgets reads the number of widgets the script asked for: a whole
// number from 0 to maxBenchWidgets, and 0 for anything else.
func benchWidgets(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > maxBenchWidgets {
		return 0
	}
	return n
}

// BenchReport writes one result line of the render bench to stderr, where the
// script reads it. It does nothing unless a bench is running.
func (a *App) BenchReport(line string) {
	if os.Getenv("DAVISH_BENCH") == "" {
		return
	}
	fmt.Fprintln(os.Stderr, benchLine(line))
}

// maxBenchLine bounds a result line; the numbers need far less.
const maxBenchLine = 400

// benchLine is line as one "davish-bench ..." line: no line breaks inside it
// and no more than maxBenchLine bytes, so a stray value cannot forge a second
// line for the script to read.
func benchLine(line string) string {
	line = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, line)
	if len(line) > maxBenchLine {
		line = line[:maxBenchLine]
	}
	return "davish-bench " + line
}
