package docker

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Image is one row of docker images.
type Image struct {
	ID         string `json:"id"` // sha256:…, full
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Created    string `json:"created"`      // as docker prints it
	CreatedAgo string `json:"createdSince"` // "2 days ago"
	Size       int64  `json:"size"`
	// Containers is how many containers, running or not, use the image.
	Containers int `json:"containers"`
	// Dangling is an untagged image no tag points at any more.
	Dangling bool `json:"dangling"`
}

// Ref is how commands name the image: repo:tag, or its ID when untagged.
func (i Image) Ref() string {
	if i.Dangling {
		return i.ID
	}
	return i.Repository + ":" + i.Tag
}

// Images lists the server's images, newest first, with how many containers
// use each. Old docker versions do not report the count, so it is taken
// from docker ps then.
func (s *Service) Images(ctx context.Context, serverID string) ([]Image, error) {
	res, err := s.read(ctx, serverID, shellx.Cmd("docker", "images", "--no-trunc", "--format", "{{json .}}"), readTimeout)
	if err != nil {
		return nil, err
	}
	if err := checkExit(res); err != nil {
		return nil, err
	}
	imgs, counted, err := parseImages(res.Stdout)
	if err != nil {
		return nil, err
	}
	if !counted {
		ps, err := s.read(ctx, serverID, shellx.Cmd("docker", "ps", "-a", "--format", "{{.Image}}"), readTimeout)
		if err == nil && ps.Exit == 0 {
			countUses(imgs, strings.Fields(string(ps.Stdout)))
		}
	}
	return imgs, nil
}

func parseImages(out []byte) ([]Image, bool, error) {
	var list []Image
	counted := true
	err := eachJSON(out, func(line []byte) error {
		var r struct {
			ID, Repository, Tag, CreatedAt, CreatedSince, Size, Containers string
		}
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		n, err := strconv.Atoi(r.Containers)
		if err != nil {
			counted = false
		}
		list = append(list, Image{
			ID:         r.ID,
			Repository: r.Repository,
			Tag:        r.Tag,
			Created:    r.CreatedAt,
			CreatedAgo: r.CreatedSince,
			Size:       size(r.Size),
			Containers: n,
			Dangling:   r.Repository == "<none>" || r.Tag == "<none>",
		})
		return nil
	})
	if err != nil {
		return nil, false, apperr.Wrap(apperr.Internal, "unexpected docker images output", err)
	}
	// "2026-10-26 09:12:40 +0700 +07" sorts by time as text within a zone.
	sort.SliceStable(list, func(i, j int) bool { return list[i].Created > list[j].Created })
	return list, counted, nil
}

// countUses counts containers per image from docker ps's Image column,
// which holds the name the container was created with, or a short ID.
func countUses(imgs []Image, used []string) {
	for i := range imgs {
		img := &imgs[i]
		img.Containers = 0
		short := strings.TrimPrefix(img.ID, "sha256:")
		for _, u := range used {
			switch {
			case !img.Dangling && (u == img.Repository+":"+img.Tag || (img.Tag == "latest" && u == img.Repository)):
				img.Containers++
			case len(u) >= 12 && strings.HasPrefix(short, u):
				img.Containers++
			}
		}
	}
}

// imageRef matches a repo[:tag], with an optional registry, or an ID.
var imageRef = regexp.MustCompile(`^(sha256:)?[a-zA-Z0-9][a-zA-Z0-9._/:@-]{0,254}$`)

// CheckImage rejects anything that is not an image reference.
func CheckImage(ref string) error {
	if !imageRef.MatchString(ref) || strings.Contains(ref, "..") {
		return apperr.New(apperr.InvalidInput, "not an image reference")
	}
	return nil
}

// Layer is one step of an image's history.
type Layer struct {
	CreatedBy string `json:"createdBy"`
	Size      int64  `json:"size"`
	Comment   string `json:"comment,omitempty"`
}

// ImageDetail is an image's layers and how it runs.
type ImageDetail struct {
	Layers       []Layer  `json:"layers"` // oldest first, as Dockerfiles read
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	Entrypoint   []string `json:"entrypoint"`
	Cmd          []string `json:"cmd"`
	WorkingDir   string   `json:"workingDir"`
	User         string   `json:"user"`
	Env          []string `json:"env"`
	ExposedPorts []string `json:"exposedPorts"`
}

const inspectSep = "--davish-inspect--"

// Image reads an image's history and config in one command.
func (s *Service) Image(ctx context.Context, serverID, ref string) (ImageDetail, error) {
	if err := CheckImage(ref); err != nil {
		return ImageDetail{}, err
	}
	line := shellx.Cmd("docker", "history", "--no-trunc", "--format", "{{json .}}", ref).String() +
		" && echo " + inspectSep + " && " +
		shellx.Cmd("docker", "image", "inspect", "--format", "{{json .}}", ref).String()
	res, err := s.read(ctx, serverID, shellx.Cmd("sh", "-c", line), readTimeout)
	if err != nil {
		return ImageDetail{}, err
	}
	if res.Exit != 0 && strings.Contains(string(res.Stderr), "No such image") {
		return ImageDetail{}, apperr.New(apperr.NotFound, "no such image")
	}
	if err := checkExit(res); err != nil {
		return ImageDetail{}, err
	}
	return parseImageDetail(res.Stdout)
}

func parseImageDetail(out []byte) (ImageDetail, error) {
	hist, insp, _ := strings.Cut(string(out), inspectSep+"\n")
	var d ImageDetail
	bad := func(err error) (ImageDetail, error) {
		return ImageDetail{}, apperr.Wrap(apperr.Internal, "unexpected docker image output", err)
	}
	err := eachJSON([]byte(hist), func(line []byte) error {
		var r struct{ CreatedBy, Size, Comment string }
		if err := json.Unmarshal(line, &r); err != nil {
			return err
		}
		d.Layers = append(d.Layers, Layer{
			CreatedBy: cleanStep(r.CreatedBy),
			Size:      size(r.Size),
			Comment:   r.Comment,
		})
		return nil
	})
	if err != nil {
		return bad(err)
	}
	// docker history lists the newest layer first.
	for i, j := 0, len(d.Layers)-1; i < j; i, j = i+1, j-1 {
		d.Layers[i], d.Layers[j] = d.Layers[j], d.Layers[i]
	}
	var r struct {
		Architecture, Os string
		Config           struct {
			Entrypoint, Cmd, Env []string
			WorkingDir, User     string
			ExposedPorts         map[string]struct{}
		}
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(insp)), &r); err != nil {
		return bad(err)
	}
	d.Architecture, d.OS = r.Architecture, r.Os
	d.Entrypoint, d.Cmd, d.Env = r.Config.Entrypoint, r.Config.Cmd, r.Config.Env
	d.WorkingDir, d.User = r.Config.WorkingDir, r.Config.User
	for p := range r.Config.ExposedPorts {
		d.ExposedPorts = append(d.ExposedPorts, p)
	}
	sort.Strings(d.ExposedPorts)
	return d, nil
}

// cleanStep turns builder noise into the Dockerfile line it came from:
// "/bin/sh -c #(nop)  CMD [\"node\"]" reads CMD ["node"].
func cleanStep(s string) string {
	s = strings.TrimSpace(s)
	if rest, ok := strings.CutPrefix(s, "/bin/sh -c #(nop) "); ok {
		return strings.TrimSpace(rest)
	}
	if rest, ok := strings.CutPrefix(s, "/bin/sh -c "); ok {
		return "RUN " + rest
	}
	return s
}

// ImagePlan is what the UI shows before removing an image.
type ImagePlan struct {
	Ref string `json:"ref"`
	// Command is exactly what RemoveImage runs.
	Command string `json:"command"`
	// Run is the docker run line Davish types into a terminal.
	Run string `json:"run"`
}

// PlanImage gives the commands for an image. Removing never forces: an
// image a container still uses stays, and docker says which one.
func PlanImage(ref string) (ImagePlan, error) {
	if err := CheckImage(ref); err != nil {
		return ImagePlan{}, err
	}
	return ImagePlan{
		Ref:     ref,
		Command: shellx.Cmd("docker", "rmi", ref).String(),
		Run:     shellx.Cmd("docker", "run", "--rm", "-it", ref).String(),
	}, nil
}

// RemoveImage runs docker rmi after the user confirmed command.
func (s *Service) RemoveImage(ctx context.Context, serverID, ref, command string) error {
	p, err := PlanImage(ref)
	if err != nil {
		return err
	}
	if command != p.Command {
		return apperr.New(apperr.InvalidInput, "command does not match what was confirmed")
	}
	res, err := s.run.Run(ctx, serverID, app, shellx.Cmd("docker", "rmi", ref))
	if err != nil {
		return err
	}
	if res.Exit != 0 {
		msg := strings.TrimSpace(string(res.Stderr))
		if strings.Contains(msg, "being used by") || strings.Contains(msg, "conflict") {
			return apperr.New(apperr.CmdFailed, msg)
		}
	}
	return checkExit(res)
}
