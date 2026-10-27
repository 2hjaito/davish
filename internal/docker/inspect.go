package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Detail is what docker inspect says about one container: a summary for
// the UI and the whole document, indented, to read as text.
type Detail struct {
	Name     string `json:"name"`
	Image    string `json:"image"`
	State    string `json:"state"`
	ExitCode int    `json:"exitCode"`
	// Error is docker's own message when the container could not start.
	Error     string `json:"error"`
	OOMKilled bool   `json:"oomKilled"`
	// Health is healthy, unhealthy or starting; empty without a check.
	Health    string   `json:"health"`
	StartedAt string   `json:"startedAt"` // RFC 3339, empty if never
	Restarts  int      `json:"restarts"`
	Policy    string   `json:"policy"` // restart policy: no, always, unless-stopped, on-failure
	Ports     []string `json:"ports"`  // "0.0.0.0:8080→80/tcp"
	Mounts    []string `json:"mounts"` // "/srv/data → /data"
	JSON      string   `json:"json"`
}

// Inspect returns the detail of one container.
func (s *Service) Inspect(ctx context.Context, serverID, name string) (Detail, error) {
	if err := CheckName(name); err != nil {
		return Detail{}, err
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("docker", "inspect", "--type", "container", name))
	if err != nil {
		return Detail{}, err
	}
	if err := checkExit(res); err != nil {
		return Detail{}, err
	}
	return parseInspect(res.Stdout)
}

func parseInspect(out []byte) (Detail, error) {
	var docs []json.RawMessage
	if err := json.Unmarshal(out, &docs); err != nil || len(docs) != 1 {
		return Detail{}, apperr.New(apperr.Internal, "unexpected docker inspect output")
	}
	var r struct {
		Name         string
		RestartCount int
		State        struct {
			Status    string
			ExitCode  int
			Error     string
			OOMKilled bool
			StartedAt string
			Health    *struct{ Status string }
		}
		Config     struct{ Image string }
		HostConfig struct {
			RestartPolicy struct{ Name string }
		}
		NetworkSettings struct {
			Ports map[string][]struct{ HostIP, HostPort string } `json:"Ports"`
		}
		Mounts []struct{ Source, Destination string }
	}
	if err := json.Unmarshal(docs[0], &r); err != nil {
		return Detail{}, apperr.Wrap(apperr.Internal, "unexpected docker inspect output", err)
	}
	d := Detail{
		Name:      strings.TrimPrefix(r.Name, "/"),
		Image:     r.Config.Image,
		State:     r.State.Status,
		ExitCode:  r.State.ExitCode,
		Error:     r.State.Error,
		OOMKilled: r.State.OOMKilled,
		Restarts:  r.RestartCount,
		Policy:    r.HostConfig.RestartPolicy.Name,
		Ports:     []string{},
		Mounts:    []string{},
	}
	// Docker writes the zero time for a container that never ran.
	if !strings.HasPrefix(r.State.StartedAt, "0001-") {
		d.StartedAt = r.State.StartedAt
	}
	if r.State.Health != nil {
		d.Health = r.State.Health.Status
	}
	for port, binds := range r.NetworkSettings.Ports {
		if len(binds) == 0 {
			d.Ports = append(d.Ports, port)
		}
		for _, b := range binds {
			d.Ports = append(d.Ports, b.HostIP+":"+b.HostPort+"→"+port)
		}
	}
	sort.Strings(d.Ports)
	for _, m := range r.Mounts {
		d.Mounts = append(d.Mounts, m.Source+" → "+m.Destination)
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, docs[0], "", "  "); err == nil {
		d.JSON = pretty.String()
	}
	return d, nil
}

// LogLines is how many recent lines Logs returns.
const LogLines = 200

// Logs is a container's latest output, stdout and stderr merged in time
// order.
type Logs struct {
	Lines []LogLine `json:"lines"`
}

// LogLine is one line of output.
type LogLine struct {
	Time   string `json:"time"` // RFC 3339 with nanoseconds, from docker
	Text   string `json:"text"`
	Stderr bool   `json:"stderr"`
}

// Logs returns the last LogLines lines the container wrote.
func (s *Service) Logs(ctx context.Context, serverID, name string) (Logs, error) {
	if err := CheckName(name); err != nil {
		return Logs{}, err
	}
	cmd := shellx.Cmd("docker", "logs", "--tail", strconv.Itoa(LogLines), "--timestamps", name)
	res, err := s.run.Run(ctx, serverID, app, cmd)
	if err != nil {
		return Logs{}, err
	}
	if err := checkExit(res); err != nil {
		return Logs{}, err
	}
	return mergeLogs(res.Stdout, res.Stderr), nil
}

// mergeLogs interleaves the two streams by their docker timestamps, which
// sort as text, and keeps the latest LogLines.
func mergeLogs(stdout, stderr []byte) Logs {
	var lines []LogLine
	add := func(out []byte, isErr bool) {
		for l := range strings.Lines(string(out)) {
			l = strings.TrimRight(l, "\r\n")
			if l == "" {
				continue
			}
			ts, text, ok := strings.Cut(l, " ")
			if !ok {
				ts, text = "", l
			}
			lines = append(lines, LogLine{Time: ts, Text: text, Stderr: isErr})
		}
	}
	add(stdout, false)
	add(stderr, true)
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	if len(lines) > LogLines {
		lines = lines[len(lines)-LogLines:]
	}
	if lines == nil {
		lines = []LogLine{}
	}
	return Logs{Lines: lines}
}
