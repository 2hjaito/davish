package docker

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// CreateObject makes a network or a volume (kind "network" or "volume").
// Creating removes nothing, so it runs without a confirmation.
func (s *Service) CreateObject(ctx context.Context, serverID, kind, name string) error {
	if kind != "network" && kind != "volume" {
		return apperr.New(apperr.InvalidInput, "unknown kind")
	}
	if !objectName.MatchString(name) {
		return apperr.New(apperr.InvalidInput, "not a name")
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("docker", kind, "create", name))
	if err != nil {
		return err
	}
	if res.Exit != 0 && strings.Contains(string(res.Stderr), "already exists") {
		return apperr.New(apperr.FSExists, "a "+kind+" with this name exists")
	}
	return checkExit(res)
}

const recreateSep = "--davish-image--"

// RunCommand rebuilds the docker run line that would make a container
// like name: its image, name, ports, environment the image does not set
// itself, mounts, restart policy, network and command.
func (s *Service) RunCommand(ctx context.Context, serverID, name string) (RunPlan, error) {
	if err := CheckName(name); err != nil {
		return RunPlan{}, err
	}
	// The image is looked up by the container's image ID, so its own
	// defaults (env, command) can be left out of the line.
	imageID := `"$(docker inspect --type container --format '{{.Image}}' ` + shellx.Quote(name) + `)"`
	line := shellx.Cmd("docker", "inspect", "--type", "container", name).String() +
		" && echo " + recreateSep + " && " +
		shellx.Cmd("docker", "inspect", "--type", "image", "--format", "{{json .Config}}").String() + " " + imageID
	res, err := s.read(ctx, serverID, shellx.Cmd("sh", "-c", line), readTimeout)
	if err != nil {
		return RunPlan{}, err
	}
	if err := checkExit(res); err != nil {
		return RunPlan{}, err
	}
	spec, err := specFromInspect(res.Stdout)
	if err != nil {
		return RunPlan{}, err
	}
	return PlanRun(spec)
}

func specFromInspect(out []byte) (RunSpec, error) {
	ctr, img, _ := strings.Cut(string(out), recreateSep+"\n")
	var docs []struct {
		Name   string
		Config struct {
			Image string
			Env   []string
			Cmd   []string
		}
		HostConfig struct {
			RestartPolicy struct{ Name string }
			NetworkMode   string
			PortBindings  map[string][]struct{ HostIp, HostPort string } //nolint:revive // docker's names
		}
		Mounts []struct {
			Type, Name, Source, Destination string
			RW                              bool
		}
	}
	var image struct {
		Env []string
		Cmd []string
	}
	if err := json.Unmarshal([]byte(ctr), &docs); err != nil || len(docs) != 1 {
		return RunSpec{}, apperr.New(apperr.Internal, "unexpected docker inspect output")
	}
	_ = json.Unmarshal([]byte(strings.TrimSpace(img)), &image)
	c := docs[0]
	spec := RunSpec{
		Name:    strings.TrimPrefix(c.Name, "/"),
		Image:   c.Config.Image,
		Restart: c.HostConfig.RestartPolicy.Name,
	}
	if m := c.HostConfig.NetworkMode; m != "" && m != "default" && m != "bridge" && objectName.MatchString(m) {
		spec.Network = m
	}
	keys := make([]string, 0, len(c.HostConfig.PortBindings))
	for k := range c.HostConfig.PortBindings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		port, proto, _ := strings.Cut(k, "/")
		cp, err := strconv.Atoi(port)
		if err != nil {
			continue
		}
		for _, b := range c.HostConfig.PortBindings[k] {
			hp, _ := strconv.Atoi(b.HostPort)
			host := b.HostIp
			if host == "" {
				// Docker binds every address when none is given.
				host = "0.0.0.0"
			}
			spec.Ports = append(spec.Ports, PortMap{Host: host, HostPort: hp, ContainerPort: cp, UDP: proto == "udp"})
		}
	}
	for _, e := range c.Config.Env {
		if !slices.Contains(image.Env, e) {
			spec.Env = append(spec.Env, e)
		}
	}
	for _, m := range c.Mounts {
		src := m.Source
		if m.Type == "volume" {
			src = m.Name
		}
		spec.Mounts = append(spec.Mounts, Mount{Source: src, Target: m.Destination, ReadOnly: !m.RW})
	}
	if !slices.Equal(c.Config.Cmd, image.Cmd) {
		spec.Args = c.Config.Cmd
	}
	return spec, nil
}
