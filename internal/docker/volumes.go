package docker

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Volume is one docker volume.
type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	Created    string `json:"created"`
	// Size is -1 when docker could not tell (another driver).
	Size int64 `json:"size"`
	// Containers is how many containers mount it, running or not.
	Containers int `json:"containers"`
	// Anonymous is a volume docker named with a hash for a container.
	Anonymous bool   `json:"anonymous"`
	Project   string `json:"project,omitempty"`
}

const volSep = "--davish-volumes--"

// volumesLine reads sizes and use counts (docker system df -v), then the
// creation times, which only docker volume inspect has.
var volumesLine = shellx.Cmd("docker", "system", "df", "-v", "--format", "json").String() +
	" && echo " + volSep + " && docker volume ls -q | xargs -r docker volume inspect --format '{{json .}}'"

// Volumes lists the server's volumes, largest first.
func (s *Service) Volumes(ctx context.Context, serverID string) ([]Volume, error) {
	res, err := s.read(ctx, serverID, shellx.Cmd("sh", "-c", volumesLine), volumeTimeout)
	if err != nil {
		return nil, err
	}
	if err := checkExit(res); err != nil {
		return nil, err
	}
	return parseVolumes(res.Stdout)
}

// hexName is how docker names an anonymous volume: 64 hex digits.
func hexName(s string) bool {
	return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == ""
}

func parseVolumes(out []byte) ([]Volume, error) {
	df, insp, _ := strings.Cut(string(out), volSep+"\n")
	var sys struct {
		Volumes []struct{ Name, Driver, Mountpoint, Links, Size, Labels string }
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(df)), &sys); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "unexpected docker system df output", err)
	}
	created := map[string]string{}
	err := eachJSON([]byte(insp), func(line []byte) error {
		var v struct{ Name, CreatedAt string }
		if err := json.Unmarshal(line, &v); err != nil {
			return err
		}
		created[v.Name] = v.CreatedAt
		return nil
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "unexpected docker volume inspect output", err)
	}
	list := make([]Volume, 0, len(sys.Volumes))
	for _, v := range sys.Volumes {
		vol := Volume{
			Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, Created: created[v.Name],
			Size: -1, Anonymous: hexName(v.Name), Project: label(v.Labels, "com.docker.compose.project"),
		}
		if v.Size != "" && v.Size != "N/A" {
			vol.Size = size(v.Size)
		}
		vol.Containers, _ = strconv.Atoi(v.Links)
		list = append(list, vol)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Size > list[j].Size })
	return list, nil
}

// VolumePlan is what the UI shows before removing a volume.
type VolumePlan struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// PlanVolume gives the command that removes a volume. Never forced: a
// volume in use stays, and docker says which container holds it.
func PlanVolume(name string) (VolumePlan, error) {
	if err := CheckName(name); err != nil {
		return VolumePlan{}, err
	}
	return VolumePlan{Name: name, Command: shellx.Cmd("docker", "volume", "rm", name).String()}, nil
}

// RemoveVolume runs docker volume rm after the user confirmed command.
func (s *Service) RemoveVolume(ctx context.Context, serverID, name, command string) error {
	p, err := PlanVolume(name)
	if err != nil {
		return err
	}
	if command != p.Command {
		return apperr.New(apperr.InvalidInput, "command does not match what was confirmed")
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("docker", "volume", "rm", name))
	if err != nil {
		return err
	}
	if res.Exit != 0 && strings.Contains(string(res.Stderr), "in use") {
		return apperr.New(apperr.CmdFailed, strings.TrimSpace(string(res.Stderr)))
	}
	return checkExit(res)
}
