package sysmon

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	"davish/internal/apperr"
)

//go:embed info.sh
var infoScript string

// SystemInfo describes the server. Fields the server does not reveal are empty.
type SystemInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"` // e.g. "Ubuntu 24.04.5 LTS"
	Kernel   string `json:"kernel"`
	Arch     string `json:"arch"`
	CPUModel string `json:"cpuModel"`
	Virt     string `json:"virt"` // kvm, docker…; empty on bare metal
}

// SystemInfo reads what the server is; it does not change while connected.
func (s *Service) SystemInfo(ctx context.Context, serverID string) (SystemInfo, error) {
	client, err := s.clients.Client(serverID)
	if err != nil {
		return SystemInfo{}, err
	}
	sections, err := runScript(ctx, client, infoScript)
	if err != nil {
		return SystemInfo{}, apperr.Wrap(apperr.Internal, "could not read system info", err)
	}
	return parseInfo(sections), nil
}

func parseInfo(s map[string][]string) SystemInfo {
	var info SystemInfo
	osRelease := map[string]string{}
	for _, line := range s["os"] {
		if k, v, ok := strings.Cut(line, "="); ok {
			if uq, err := strconv.Unquote(v); err == nil {
				v = uq
			}
			osRelease[k] = strings.Trim(v, `'"`)
		}
	}
	info.OS = osRelease["PRETTY_NAME"]
	if info.OS == "" {
		info.OS = strings.TrimSpace(osRelease["NAME"] + " " + osRelease["VERSION"])
	}
	if u := s["uname"]; len(u) >= 2 {
		info.Kernel, info.Arch = strings.TrimSpace(u[0]), strings.TrimSpace(u[1])
	}
	info.Hostname = firstLine(s["hostname"])
	if m := firstLine(s["cpu"]); m != "-" {
		info.CPUModel = m
	}
	if v := firstLine(s["virt"]); v != "none" {
		info.Virt = v
	}
	return info
}

func firstLine(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[0])
}
