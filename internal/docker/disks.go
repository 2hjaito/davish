package docker

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"davish/internal/shellx"
)

// DiskUse is where Docker keeps its data and what each container mounts,
// for Monitor's Disks card to say which drive holds which container's data.
type DiskUse struct {
	Version string `json:"version"`
	// RootDir holds images, containers and named volumes.
	RootDir string           `json:"rootDir"`
	Mounts  []ContainerMount `json:"mounts"`
}

// ContainerMount is one volume or bind mount of a container.
type ContainerMount struct {
	Container string `json:"container"`
	Project   string `json:"project,omitempty"`
	// Type is volume or bind; tmpfs and npipe are left out.
	Type   string `json:"type"`
	Name   string `json:"name,omitempty"` // a volume's name
	Source string `json:"source"`         // the path on the server
	Dest   string `json:"dest"`
}

// infoFormat prints the version and root folder on one line.
const infoFormat = "{{.ServerVersion}}\t{{.DockerRootDir}}"

// mountsFormat prints one line per container: name, compose project, mounts.
const mountsFormat = `{{.Name}}` + "\t" + `{{index .Config.Labels "com.docker.compose.project"}}` + "\t" + `{{json .Mounts}}`

// DiskUse reads docker info and the mounts of every container, running or
// not, in two read-only commands that go to the command log like the rest.
func (s *Service) DiskUse(ctx context.Context, serverID string) (DiskUse, error) {
	res, err := s.read(ctx, serverID, shellx.Cmd("docker", "info", "--format", infoFormat), readTimeout)
	if err != nil {
		return DiskUse{}, err
	}
	if err := checkExit(res); err != nil {
		return DiskUse{}, err
	}
	out := DiskUse{Mounts: []ContainerMount{}}
	out.Version, out.RootDir, _ = strings.Cut(strings.TrimSpace(string(res.Stdout)), "\t")

	ids, err := s.read(ctx, serverID, shellx.Cmd("docker", "ps", "-aq", "--no-trunc"), readTimeout)
	if err != nil {
		return DiskUse{}, err
	}
	if err := checkExit(ids); err != nil {
		return DiskUse{}, err
	}
	list := strings.Fields(string(ids.Stdout))
	if len(list) == 0 {
		return out, nil
	}
	args := append([]string{"inspect", "--format", mountsFormat}, list...)
	res, err = s.read(ctx, serverID, shellx.Cmd("docker", args...), readTimeout)
	if err != nil {
		return DiskUse{}, err
	}
	// A container removed between ps and inspect fails the command but the
	// others are still printed.
	out.Mounts = parseMounts(res.Stdout)
	return out, nil
}

func parseMounts(b []byte) []ContainerMount {
	list := []ContainerMount{}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		var ms []struct {
			Type, Name, Source, Destination string
		}
		if json.Unmarshal([]byte(parts[2]), &ms) != nil {
			continue
		}
		project := parts[1]
		if project == "<no value>" {
			project = ""
		}
		for _, m := range ms {
			if (m.Type != "volume" && m.Type != "bind") || m.Source == "" {
				continue
			}
			list = append(list, ContainerMount{
				Container: strings.TrimPrefix(parts[0], "/"), Project: project,
				Type: m.Type, Name: m.Name, Source: m.Source, Dest: m.Destination,
			})
		}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Container < list[j].Container })
	return list
}
