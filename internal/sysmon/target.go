package sysmon

import (
	"fmt"

	"davish/internal/platform"
)

// target is what Monitor runs on one kind of server: how a script reaches
// the remote shell and which scripts read the metrics.
type target struct {
	// launch returns the command a session starts and the stdin it gets.
	launch func(script string) (command, stdin string)
	// vars is the script's first lines: the pace, in the shell's syntax.
	vars func(every, healthEvery int) string
	// collect prints metric snapshots until it is stopped; procs and info
	// print one answer and exit.
	collect, procs, info string
}

// posixLaunch feeds the script to sh on stdin, so no command line is built
// and the user's login shell (fish, csh…) does not matter.
func posixLaunch(script string) (string, string) { return "sh -s", script }

func posixVars(every, healthEvery int) string {
	return fmt.Sprintf("every=%d\nhealthEvery=%d\n", every, healthEvery)
}

func powershellVars(every, healthEvery int) string {
	return fmt.Sprintf("$every = %d\n$healthEvery = %d\n", every, healthEvery)
}

// targetFor returns the scripts for a server's OS, or false when Monitor
// cannot read that kind of server.
//
// Unknown means detection timed out or gave an answer nobody knows; it
// takes the Linux scripts, since a Linux box with a slow login shell is the
// likeliest cause. If they cannot read it, the collector reports an error
// state instead of waiting for ever.
func targetFor(os platform.OS) (target, bool) {
	switch os {
	case platform.Linux, platform.Unknown, "":
		return target{launch: posixLaunch, vars: posixVars, collect: collectScript, procs: procsScript, info: infoScript}, true
	case platform.Windows:
		// The scripts print the same @@sections as the Linux ones (in the
		// shapes of /proc/stat, /proc/meminfo, df -kP…), so one parser reads both.
		return target{launch: platform.PowerShell, vars: powershellVars, collect: collectWinScript, procs: procsWinScript, info: infoWinScript}, true
	}
	return target{}, false
}
