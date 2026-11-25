//go:build !windows

package conn

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	shellProbeTimeout = 3 * time.Second
	shellAgentStart   = "__DAVISH_SSH_AUTH_SOCK_START__"
	shellAgentEnd     = "__DAVISH_SSH_AUTH_SOCK_END__"
	shellAgentCmd     = `printf '\n` + shellAgentStart + `%s` + shellAgentEnd + `\n' "$SSH_AUTH_SOCK"`
)

var (
	agentEnvOnce sync.Once
	agentEnvDone atomic.Pointer[chan struct{}]
)

// InitAgentEnv resolves SSH_AUTH_SOCK from the user's login shell when Davish
// is launched as a desktop GUI app outside a terminal (for example, Davish.app
// from Finder or Spotlight on macOS, where launchd sets its own default socket
// instead of running ~/.zshrc). It runs in the background so startup is not
// delayed; dialAgent waits for it if a connection starts before it finishes.
func InitAgentEnv(log *slog.Logger) {
	agentEnvOnce.Do(func() {
		sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
		if !shouldProbeShellAgent(
			runtime.GOOS,
			os.Getppid(),
			os.Getenv("TERM"),
			os.Getenv("DAVISH_BENCH"),
			os.Getenv("DISPLAY"),
			os.Getenv("WAYLAND_DISPLAY"),
			sock,
		) {
			return
		}
		done := make(chan struct{})
		agentEnvDone.Store(&done)
		go func() {
			defer close(done)
			ctx, cancel := context.WithTimeout(context.Background(), shellProbeTimeout)
			defer cancel()
			shell := userShell(runtime.GOOS, os.Getenv("SHELL"))
			resolved, err := probeShellAgentSock(ctx, shell)
			if err != nil {
				if log != nil {
					log.Info("ssh agent shell env probe failed", "shell", shell, "err", err)
				}
				return
			}
			if resolved != "" && resolved != sock {
				_ = os.Setenv("SSH_AUTH_SOCK", resolved)
				if log != nil {
					log.Info("ssh agent env resolved from shell", "shell", shell, "sock", resolved)
				}
			}
		}()
	})
}

func waitAgentEnv() {
	if p := agentEnvDone.Load(); p != nil {
		<-*p
	}
}

func shouldProbeShellAgent(goos string, ppid int, term, bench, display, wayland, sock string) bool {
	if bench != "" || term != "" {
		return false
	}
	if goos == "darwin" {
		return ppid == 1 && (sock == "" || isLaunchdAgentSock(sock))
	}
	return display != "" || wayland != ""
}

// isLaunchdAgentSock reports whether sock is macOS launchd's built-in agent
// socket, which GUI apps inherit instead of an SSH_AUTH_SOCK set in ~/.zshrc.
func isLaunchdAgentSock(sock string) bool {
	return (strings.HasPrefix(sock, "/private/tmp/com.apple.launchd.") ||
		strings.HasPrefix(sock, "/tmp/com.apple.launchd.")) &&
		strings.HasSuffix(sock, "/Listeners")
}

func userShell(goos, envShell string) string {
	if s := strings.TrimSpace(envShell); filepath.IsAbs(s) {
		return s
	}
	if goos == "darwin" {
		return "/bin/zsh"
	}
	return "/bin/sh"
}

func probeShellAgentSock(ctx context.Context, shell string) (string, error) {
	if !filepath.IsAbs(shell) {
		return "", errors.New("shell path is not absolute")
	}
	out, err := runShellProbe(ctx, shell, "-l", "-i", "-c", shellAgentCmd)
	if sock := parseShellAgentOutput(out); sock != "" {
		return sock, nil
	}
	// Fall back to a non-interactive login shell if interactive rc files failed.
	if err != nil && ctx.Err() == nil {
		if out2, err2 := runShellProbe(ctx, shell, "-l", "-c", shellAgentCmd); err2 == nil {
			if sock := parseShellAgentOutput(out2); sock != "" {
				return sock, nil
			}
		}
	}
	return "", err
}

func runShellProbe(ctx context.Context, shell string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, shell, args...) //nolint:gosec // user's login shell with fixed read-only printf command
	cmd.Stdin = nil
	cmd.WaitDelay = 200 * time.Millisecond
	return cmd.Output()
}

func parseShellAgentOutput(out []byte) string {
	s := string(out)
	end := strings.LastIndex(s, shellAgentEnd)
	if end < 0 {
		return ""
	}
	start := strings.LastIndex(s[:end], shellAgentStart)
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(s[start+len(shellAgentStart) : end])
}

// dialAgent connects to the agent named by SSH_AUTH_SOCK.
func dialAgent() (io.ReadWriteCloser, error) {
	waitAgentEnv()
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	if sock == "" {
		return nil, errors.New("SSH_AUTH_SOCK is not set")
	}
	return net.DialTimeout("unix", expandHome(sock), agentWait()) //nolint:gosec // SSH_AUTH_SOCK is how the user's agent is found
}
