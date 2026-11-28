//go:build windows

package conn

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/sys/windows"
)

// The test binary doubles as the relay, the way Davish.exe does.
func TestMain(m *testing.M) {
	if RunAgentProxy(os.Args) {
		return
	}
	os.Exit(m.Run())
}

// servePipeAgent listens on a fresh named pipe and answers as an SSH agent
// holding one key, for every client until the test ends.
func servePipeAgent(t *testing.T) (pipe string, pub ssh.PublicKey) {
	t.Helper()
	pipe = fmt.Sprintf(`\\.\pipe\davish-test-agent-%d-%d`, os.Getpid(), time.Now().UnixNano())
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kr := agent.NewKeyring()
	if err := kr.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(pipe)
	if err != nil {
		t.Fatal(err)
	}
	newInstance := func() (windows.Handle, error) {
		return windows.CreateNamedPipe(name,
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
			windows.PIPE_UNLIMITED_INSTANCES, 4096, 4096, 0, nil)
	}
	h, err := newInstance()
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		for {
			err := windows.ConnectNamedPipe(h, nil)
			if err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
				_ = windows.CloseHandle(h)
				return
			}
			c := os.NewFile(uintptr(h), pipe)
			go func() { _ = agent.ServeAgent(kr, c); _ = c.Close() }()
			select {
			case <-stop:
				return
			default:
			}
			if h, err = newInstance(); err != nil {
				return
			}
		}
	}()
	return pipe, signer.PublicKey()
}

// Through the relay, the agent lists its key and signs with it (#332).
func TestAgentPipeThroughRelay(t *testing.T) {
	pipe, pub := servePipeAgent(t)
	c, err := dialAgentPipe(pipe)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, ok := c.(*agentProxy); !ok {
		t.Fatalf("pipe opened as %T, not through the relay", c)
	}
	signers, err := agent.NewClient(c).Signers()
	if err != nil {
		t.Fatal(err)
	}
	if len(signers) != 1 || string(signers[0].PublicKey().Marshal()) != string(pub.Marshal()) {
		t.Fatalf("got %d keys, want the test key", len(signers))
	}
	data := []byte("session")
	sig, err := signers[0].Sign(rand.Reader, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.Verify(data, sig); err != nil {
		t.Fatal(err)
	}
}

// The relay is a child of the app, which is the whole point.
func TestAgentRelayIsChild(t *testing.T) {
	pipe, _ := servePipeAgent(t)
	c, err := dialAgentPipe(pipe)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	p := c.(*agentProxy)
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if int(e.ProcessID) == p.cmd.Process.Pid {
			if int(e.ParentProcessID) != os.Getpid() {
				t.Fatalf("relay's parent is %d, want %d", e.ParentProcessID, os.Getpid())
			}
			return
		}
	}
	t.Fatal("relay process not found")
}

// A pipe nobody listens on is an error from the relay, not a fallback that
// would hide it.
func TestAgentRelayMissingPipe(t *testing.T) {
	_, err := dialAgentPipe(`\\.\pipe\davish-test-no-agent`)
	if err == nil {
		t.Fatal("no error for a missing pipe")
	}
	if errors.Is(err, errProxyStart) {
		t.Fatalf("relay did not start: %v", err)
	}
}

// When the relay cannot start, the pipe is opened directly as before.
func TestAgentRelayFallback(t *testing.T) {
	pipe, _ := servePipeAgent(t)
	old := agentProxyExe
	agentProxyExe = func() (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { agentProxyExe = old })
	c, err := dialAgentPipe(pipe)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, ok := c.(*os.File); !ok {
		t.Fatalf("got %T, want the pipe opened directly", c)
	}
	if _, err := agent.NewClient(c).List(); err != nil {
		t.Fatal(err)
	}
}

func TestRunAgentProxyIgnoresOtherArgs(t *testing.T) {
	for _, args := range [][]string{{"Davish.exe"}, {"Davish.exe", AgentProxyFlag}, {"Davish.exe", "--other", `\\.\pipe\x`}} {
		if RunAgentProxy(args) {
			t.Fatalf("RunAgentProxy(%q) took over", args)
		}
	}
}
