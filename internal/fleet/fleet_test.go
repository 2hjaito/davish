package fleet

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/events"
)

// fakeTerms runs every session until its server is released, or closed.
type fakeTerms struct {
	mu      sync.Mutex
	running int
	peak    int
	exits   map[string]int // serverID → exit code; missing waits for Close
	fail    map[string]error
	open    map[string]chan int
	started []string
}

func newFake() *fakeTerms {
	return &fakeTerms{exits: map[string]int{}, fail: map[string]error{}, open: map[string]chan int{}}
}

func (f *fakeTerms) Run(_ context.Context, serverID string, _, _ int, _ string, onStart func(string)) (string, <-chan int, error) {
	f.mu.Lock()
	if err := f.fail[serverID]; err != nil {
		f.mu.Unlock()
		return "", nil, err
	}
	f.running++
	if f.running > f.peak {
		f.peak = f.running
	}
	f.started = append(f.started, serverID)
	inner := make(chan int, 1)
	tid := "t-" + serverID
	f.open[tid] = inner
	code, ok := f.exits[serverID]
	f.mu.Unlock()

	onStart(tid)
	done := make(chan int, 1)
	go func() {
		if ok {
			time.Sleep(10 * time.Millisecond)
			inner <- code
		}
		c := <-inner
		f.mu.Lock()
		f.running--
		f.mu.Unlock()
		done <- c
	}()
	return tid, done, nil
}

func (f *fakeTerms) Close(id string) error {
	f.mu.Lock()
	ch := f.open[id]
	f.mu.Unlock()
	select {
	case ch <- -1:
	default:
	}
	return nil
}

type recorder struct {
	mu   sync.Mutex
	jobs []Job
	all  chan Job
}

func (r *recorder) Emit(name string, payload any) {
	if name != events.FleetJob {
		return
	}
	j := payload.(Job)
	r.mu.Lock()
	r.jobs = append(r.jobs, j)
	r.mu.Unlock()
	r.all <- j
}

type auditLog struct {
	mu      sync.Mutex
	entries []audit.Entry
}

func (a *auditLog) Add(e audit.Entry) audit.Entry {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, e)
	return e
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// final waits until every server has a final state and returns them.
func final(t *testing.T, rec *recorder, n int) map[string]Job {
	t.Helper()
	out := map[string]Job{}
	timeout := time.After(5 * time.Second)
	for len(out) < n {
		select {
		case j := <-rec.all:
			if j.State != Running && j.State != Queued {
				out[j.ServerID] = j
			}
		case <-timeout:
			t.Fatalf("only %d of %d finished: %+v", len(out), n, out)
		}
	}
	return out
}

func items(n int, cmd string) []Item {
	var it []Item
	for i := range n {
		it = append(it, Item{ServerID: "s" + strconv.Itoa(i), Command: cmd})
	}
	return it
}

func TestRunParallelAndAudit(t *testing.T) {
	f := newFake()
	for i := range 9 {
		f.exits["s"+strconv.Itoa(i)] = i % 3 // s0, s3, s6 succeed
	}
	rec := &recorder{all: make(chan Job, 100)}
	log := &auditLog{}
	s := New(f, log, rec, quiet())

	id, jobs, err := s.Start(context.Background(), items(9, "uptime"), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 9 || jobs[0].State != Queued || jobs[0].RunID != id {
		t.Fatalf("jobs = %+v", jobs)
	}
	got := final(t, rec, 9)
	for sid, j := range got {
		n, _ := strconv.Atoi(sid[1:])
		want := Failed
		if n%3 == 0 {
			want = Done
		}
		if j.State != want || j.Exit != n%3 || j.TermID != "t-"+sid {
			t.Errorf("%s = %+v, want %s", sid, j, want)
		}
	}
	if f.peak > Parallel {
		t.Errorf("peak %d sessions, limit %d", f.peak, Parallel)
	}
	// Servers start in turn: the first four before any other.
	head := append([]string(nil), f.started[:Parallel]...)
	sort.Strings(head)
	if !reflect.DeepEqual(head, []string{"s0", "s1", "s2", "s3"}) {
		t.Errorf("order = %v", f.started)
	}

	// Running came before the end, with the session ID.
	rec.mu.Lock()
	first := map[string]Job{}
	for _, j := range rec.jobs {
		if _, ok := first[j.ServerID]; !ok {
			first[j.ServerID] = j
		}
	}
	rec.mu.Unlock()
	if j := first["s4"]; j.State != Running || j.TermID != "t-s4" {
		t.Errorf("first event of s4 = %+v", j)
	}

	log.mu.Lock()
	defer log.mu.Unlock()
	if len(log.entries) != 9 {
		t.Fatalf("audit = %d entries", len(log.entries))
	}
	for _, e := range log.entries {
		if e.App != "servers" || e.Command != "uptime" {
			t.Errorf("audit entry %+v", e)
		}
	}
}

func TestStartFailure(t *testing.T) {
	f := newFake()
	f.exits["s0"] = 0
	f.fail["s1"] = apperr.New(apperr.ConnNotConnected, "down")
	rec := &recorder{all: make(chan Job, 10)}
	log := &auditLog{}
	s := New(f, log, rec, quiet())
	if _, _, err := s.Start(context.Background(), items(2, "uptime"), 80, 24); err != nil {
		t.Fatal(err)
	}
	got := final(t, rec, 2)
	if j := got["s1"]; j.State != Failed || j.Error != "CONN_NOT_CONNECTED" || j.Exit != -1 {
		t.Errorf("s1 = %+v", j)
	}
	if got["s0"].State != Done {
		t.Errorf("s0 = %+v", got["s0"])
	}
	if len(log.entries) != 2 {
		t.Errorf("audit = %+v", log.entries)
	}
}

func TestStop(t *testing.T) {
	f := newFake() // no exit codes: every session waits for Close
	rec := &recorder{all: make(chan Job, 100)}
	s := New(f, &auditLog{}, rec, quiet())
	id, _, err := s.Start(context.Background(), items(6, "sleep 100"), 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the first four to run.
	for n := 0; n < Parallel; {
		if j := <-rec.all; j.State == Running {
			n++
		}
	}
	s.Stop(id)
	got := final(t, rec, 6)
	var states []string
	for _, j := range got {
		states = append(states, j.State)
	}
	sort.Strings(states)
	if !reflect.DeepEqual(states, []string{Stopped, Stopped, Stopped, Stopped, Stopped, Stopped}) {
		t.Errorf("states = %v", states)
	}
	if len(f.started) != Parallel {
		t.Errorf("started %v; queued servers must not start", f.started)
	}
}

func TestValidate(t *testing.T) {
	s := New(newFake(), &auditLog{}, &recorder{all: make(chan Job, 1)}, quiet())
	bad := [][]Item{
		nil,
		{{ServerID: "a", Command: " "}},
		{{ServerID: "a", Command: "x"}, {ServerID: "a", Command: "y"}},
		{{ServerID: "a", Command: "a\x00b"}},
		items(MaxServers+1, "x"),
	}
	for i, it := range bad {
		if _, _, err := s.Start(context.Background(), it, 80, 24); err == nil {
			t.Errorf("case %d: nil error", i)
		}
	}
}

func TestDestructive(t *testing.T) {
	yes := []string{
		"rm -rf /tmp/x", "sudo rm file", "sudo -n reboot", "cd /x && rm a", "shutdown -h now",
		"systemctl restart nginx", "sudo systemctl --now disable ufw", "service nginx stop",
		"docker rm web", "docker system prune -a", "docker volume rm data", "echo hi > /etc/motd",
		"kill -9 1", "mkfs.ext4 /dev/sdb", "crontab -r", "truncate -s0 log",
	}
	no := []string{
		"uptime", "df -h", "systemctl status nginx", "docker ps", "ls 2>/dev/null",
		"echo hi >> log", "cat x > /dev/null", "grep rm file", "apt-get update", "echo 1 2>&1",
		"journalctl -u rmate", "docker logs web",
	}
	for _, c := range yes {
		if !Destructive(c) {
			t.Errorf("Destructive(%q) = false", c)
		}
	}
	for _, c := range no {
		if Destructive(c) {
			t.Errorf("Destructive(%q) = true", c)
		}
	}
	if c := CheckCommand("uptime", 2); c.Typed || c.Destructive {
		t.Errorf("2 servers = %+v", c)
	}
	if c := CheckCommand("uptime", 3); !c.Typed {
		t.Errorf("3 servers = %+v", c)
	}
	if c := CheckCommand("reboot", 1); !c.Typed || !c.Destructive {
		t.Errorf("reboot = %+v", c)
	}
}

type detector map[string]string

func (d detector) Detect(_ context.Context, id string) (string, error) {
	if id == "down" {
		return "", apperr.New(apperr.ConnNotConnected, "down")
	}
	return d[id], nil
}

func TestPlanUpdate(t *testing.T) {
	d := detector{"a": "apt", "r": "dnf", "k": "apk", "none": ""}
	u := PlanUpdate(context.Background(), d, []string{"a", "r", "k", "none", "down"})
	want := []Item{
		{ServerID: "a", Command: "sudo -n apt-get update && sudo -n apt-get upgrade"},
		{ServerID: "r", Command: "sudo -n dnf upgrade"},
		{ServerID: "k", Command: "sudo -n apk update && sudo -n apk upgrade"},
	}
	if !reflect.DeepEqual(u.Items, want) {
		t.Errorf("items = %+v", u.Items)
	}
	wantSkip := []Skip{{ServerID: "none", Reason: "noManager"}, {ServerID: "down", Reason: "CONN_NOT_CONNECTED"}}
	if !reflect.DeepEqual(u.Skipped, wantSkip) {
		t.Errorf("skipped = %+v", u.Skipped)
	}
}
