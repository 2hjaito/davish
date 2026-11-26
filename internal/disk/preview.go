package disk

import (
	"bufio"
	"bytes"
	"context"
	"slices"
	"strconv"
	"strings"

	"davish/internal/apperr"
	"davish/internal/audit"
)

// previewLimit is how many files a preview lists, the largest first.
const previewLimit = 200

// listBySize turns a find selector into a read-only listing of what it
// selects: "bytes<tab>path", the largest first. It runs nothing that
// changes the server.
func listBySize(sel string) string {
	return `export LC_ALL=C; ` + sel + ` -printf '%s\t%p\n' 2>/dev/null | sort -rn | head -n ` + strconv.Itoa(previewLimit)
}

// previews are the read-only listings of what a cleanup would remove, by
// ID. Cleanups run by a tool of their own (docker prune, apt, journalctl)
// have none: the tool decides what goes.
var previews = map[string]string{
	"logs":        listBySize(selLogs),
	"docker-logs": listBySize(selDockerLogs),
	"coredumps":   listBySize(selCoredumps),
	"tmp":         listBySize(selTmp),
	// Docker decides what its prune removes; these list the same things by
	// the filter prune uses, without sizes (-1: unknown).
	"docker-containers": `docker container ls -a --filter status=exited --filter status=created --filter status=dead` +
		` --format '{{.Names}}  ({{.Image}}, {{.Status}})' 2>/dev/null | head -n ` + strconv.Itoa(previewLimit) + ` | awk '{print "-1\t" $0}'`,
	"docker-volumes": `docker volume ls --filter dangling=true --format '{{.Name}}' 2>/dev/null | head -n ` +
		strconv.Itoa(previewLimit) + ` | awk '{print "-1\t" $0}'`,
	// The whole folder goes, so list what is in it, by size.
	"home-cache": `export LC_ALL=C; du -sk "$HOME/.cache"/* "$HOME/.cache"/.[!.]* 2>/dev/null | sort -rn | head -n ` +
		strconv.Itoa(previewLimit) + ` | awk -F'\t' '{printf "%d\t%s\n", $1*1024, $2}'`,
}

// PreviewFile is one file or folder a cleanup would remove.
type PreviewFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Preview lists what a cleanup would remove, before it runs.
type Preview struct {
	// Files are the largest first, at most previewLimit.
	Files []PreviewFile `json:"files"`
	// Root is set when the list was read as root. When false for a cleanup
	// that runs as root, folders only root can read are missing from it.
	Root bool `json:"root"`
}

// HasPreview tells whether the cleanup id can be listed.
func HasPreview(id string) bool {
	_, ok := previews[id]
	return ok
}

// Preview lists, without removing anything, what the cleanup id would
// remove: the same selection its command makes.
func (s *Service) Preview(ctx context.Context, serverID, id string) (Preview, error) {
	line, ok := previews[id]
	if !ok {
		return Preview{}, apperr.New(apperr.InvalidInput, "this cleanup cannot be listed")
	}
	var res audit.Result
	asRoot := false
	if slices.Contains(rootCleanups, id) {
		r, ok, _, err := s.measureAsRoot(ctx, serverID, line)
		if err != nil {
			return Preview{}, err
		}
		res, asRoot = r, ok
	}
	if !asRoot {
		var err error
		if res, err = s.run.RunLine(ctx, serverID, app, line); err != nil {
			return Preview{}, err
		}
	}
	return Preview{Files: parsePreview(res.Stdout), Root: asRoot}, nil
}

func parsePreview(out []byte) []PreviewFile {
	var fs []PreviewFile
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		size, path, ok := strings.Cut(sc.Text(), "\t")
		if !ok || path == "" {
			continue
		}
		n, err := strconv.ParseInt(strings.TrimSpace(size), 10, 64)
		if err != nil {
			continue
		}
		fs = append(fs, PreviewFile{Path: path, Size: n})
	}
	return fs
}
