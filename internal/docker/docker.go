// Package docker lists, inspects and controls containers through the
// docker CLI on the server. The Docker socket is never forwarded.
package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"davish/internal/apperr"
	"davish/internal/audit"
	"davish/internal/shellx"
)

// app is the name commands are recorded under in the audit log.
const app = "docker"

// Runner runs a command on a server and records it in the audit log.
type Runner interface {
	Run(ctx context.Context, serverID, app string, cmd shellx.Command) (audit.Result, error)
}

// Container is one row of docker ps -a.
type Container struct {
	ID    string `json:"id"` // full ID
	Name  string `json:"name"`
	Image string `json:"image"`
	// State is created, running, restarting, paused, exited, removing or dead.
	State string `json:"state"`
	// Status is docker's own summary: "Up 3 hours", "Exited (1) 2 minutes ago".
	Status  string `json:"status"`
	Ports   string `json:"ports"`
	Command string `json:"command"`
	Created string `json:"created"`
	// RunningFor is docker's "6 days ago": since it was created.
	RunningFor string `json:"runningFor"`
	// Project and Service come from docker compose's labels, empty for a
	// container started by hand.
	Project string `json:"project,omitempty"`
	Service string `json:"service,omitempty"`
}

// readTimeout bounds a read: a hung docker daemon must not leave the app
// waiting for ever. Volumes need longer, docker system df -v adds up sizes.
const (
	readTimeout   = 30 * time.Second
	volumeTimeout = 90 * time.Second
)

// read runs a command that only reads, within limit.
func (s *Service) read(ctx context.Context, serverID string, cmd shellx.Command, limit time.Duration) (audit.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	res, err := s.run.Run(ctx, serverID, app, cmd)
	if errors.Is(err, context.DeadlineExceeded) {
		return res, apperr.New(apperr.CmdTimeout, "docker did not answer in time")
	}
	return res, err
}

// Service talks to the docker CLI on each server.
type Service struct {
	run Runner
}

// New creates the service.
func New(run Runner) *Service {
	return &Service{run: run}
}

// List returns every container, running or not, sorted by name.
func (s *Service) List(ctx context.Context, serverID string) ([]Container, error) {
	res, err := s.read(ctx, serverID, shellx.Cmd("docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}"), readTimeout)
	if err != nil {
		return nil, err
	}
	if err := checkExit(res); err != nil {
		return nil, err
	}
	return parsePS(res.Stdout)
}

func parsePS(out []byte) ([]Container, error) {
	var list []Container
	err := eachJSON(out, func(line []byte) error {
		var r struct {
			ID, Names, Image, State, Status, Ports, Command, CreatedAt, RunningFor, Labels string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		list = append(list, Container{
			ID:         r.ID,
			Name:       strings.Split(r.Names, ",")[0],
			Image:      r.Image,
			State:      r.State,
			Status:     r.Status,
			Ports:      r.Ports,
			Command:    strings.Trim(r.Command, `"`),
			Created:    r.CreatedAt,
			RunningFor: r.RunningFor,
			Project:    label(r.Labels, "com.docker.compose.project"),
			Service:    label(r.Labels, "com.docker.compose.service"),
		})
		return nil
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "unexpected docker ps output", err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

// label reads key from docker's "k=v,k=v" label list.
func label(labels, key string) string {
	for kv := range strings.SplitSeq(labels, ",") {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v
		}
	}
	return ""
}

// eachJSON calls fn for every non-empty line of out.
func eachJSON(out []byte, fn func(line []byte) error) error {
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if err := fn(line); err != nil {
			return err
		}
	}
	return sc.Err()
}

// Stats is a running container's resource use, from docker stats.
type Stats struct {
	ID       string  `json:"id"`
	CPU      float64 `json:"cpu"` // percent of one core, can pass 100
	MemUsed  int64   `json:"memUsed"`
	MemLimit int64   `json:"memLimit"`
	PIDs     int     `json:"pids"`
}

// Stats samples every running container once. docker stats waits for a
// second sample to compute CPU, so this takes about two seconds; the UI
// asks for it apart from List.
func (s *Service) Stats(ctx context.Context, serverID string) ([]Stats, error) {
	res, err := s.read(ctx, serverID, shellx.Cmd("docker", "stats", "--no-stream", "--no-trunc", "--format", "{{json .}}"), readTimeout)
	if err != nil {
		return nil, err
	}
	if err := checkExit(res); err != nil {
		return nil, err
	}
	return parseStats(res.Stdout)
}

func parseStats(out []byte) ([]Stats, error) {
	list := []Stats{}
	err := eachJSON(out, func(line []byte) error {
		var r struct{ ID, CPUPerc, MemUsage, PIDs string }
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		st := Stats{ID: r.ID, CPU: percent(r.CPUPerc)}
		used, limit, _ := strings.Cut(r.MemUsage, "/")
		st.MemUsed = size(used)
		st.MemLimit = size(limit)
		st.PIDs, _ = strconv.Atoi(strings.TrimSpace(r.PIDs))
		list = append(list, st)
		return nil
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "unexpected docker stats output", err)
	}
	return list, nil
}

func percent(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), "%"), 64)
	return v
}

var sizeRE = regexp.MustCompile(`^([0-9.]+)\s*([A-Za-z]*)$`)

var sizeUnits = map[string]float64{
	"": 1, "b": 1,
	"kb": 1e3, "mb": 1e6, "gb": 1e9, "tb": 1e12,
	"kib": 1 << 10, "mib": 1 << 20, "gib": 1 << 30, "tib": 1 << 40,
}

// size reads docker's human sizes: "1.004MiB", "7.749GiB", "0B", "12kB".
func size(s string) int64 {
	m := sizeRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	mult, ok := sizeUnits[strings.ToLower(m[2])]
	if err != nil || !ok {
		return 0
	}
	return int64(v * mult)
}

// checkExit maps a failed docker command to an error code.
func checkExit(res audit.Result) error {
	if res.Exit == 0 {
		return nil
	}
	if res.Exit == 127 {
		return apperr.New(apperr.CmdMissing, "docker is not installed on the server")
	}
	msg := string(res.Stderr)
	switch {
	// Not in the docker group: the socket is root's.
	case strings.Contains(msg, "permission denied") && strings.Contains(msg, "docker"):
		return apperr.New(apperr.CmdNeedsRoot, "the docker socket requires root")
	case strings.Contains(msg, "No such container"):
		return apperr.New(apperr.NotFound, "no such container")
	case strings.Contains(msg, "Cannot connect to the Docker daemon"), strings.Contains(msg, "Is the docker daemon running"):
		return apperr.New(apperr.DockerDown, "the docker daemon is not running")
	}
	return apperr.New(apperr.CmdFailed, "command failed")
}
