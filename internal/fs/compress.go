package fs

import (
	"context"
	"errors"
	"os"
	"path"
	"strconv"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/shellx"
)

// Archive formats. GZ is a gzipped tar, which works for folders as well as
// files and is what people mean by a .gz of a folder.
const (
	FormatZip   = "zip"
	FormatTarGz = "tar.gz"
	FormatTar   = "tar"
)

// CompressPlan is what compressing will run, shown before it starts.
type CompressPlan struct {
	// Output is the archive's path, next to the source, never an existing name.
	Output  string `json:"output"`
	Command string `json:"command"`
	// Sudo runs the same archiver with sudo, for a terminal when the
	// folder is not writable.
	Sudo string `json:"sudo"`
}

// PlanCompress describes how p would be archived as format.
func (s *Service) PlanCompress(_ context.Context, serverID, p, format string) (CompressPlan, error) {
	if err := CheckPath(p); err != nil {
		return CompressPlan{}, err
	}
	if p == "/" {
		return CompressPlan{}, apperr.New(apperr.InvalidInput, "cannot archive the root directory")
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return CompressPlan{}, err
	}
	if _, err := sc.Lstat(p); err != nil {
		return CompressPlan{}, mapErr("compress", err)
	}
	dir, name := path.Dir(p), path.Base(p)
	out, err := freeArchiveName(sc, dir, name, format)
	if err != nil {
		return CompressPlan{}, err
	}
	cmd, err := compressCommand(format, out, name)
	if err != nil {
		return CompressPlan{}, err
	}
	return CompressPlan{
		Output:  path.Join(dir, out),
		Command: shellx.InDir(dir, cmd),
		Sudo:    shellx.InDir(dir, shellx.Cmd("sudo", cmd.Args()...)),
	}, nil
}

func compressCommand(format, out, name string) (shellx.Command, error) {
	switch format {
	case FormatZip:
		// -y keeps links as links instead of copying what they point to.
		return shellx.Cmd("zip", "-r", "-q", "-y", out, "--", name), nil
	case FormatTarGz:
		return shellx.Cmd("tar", "-czf", out, "--", name), nil
	case FormatTar:
		return shellx.Cmd("tar", "-cf", out, "--", name), nil
	}
	return shellx.Command{}, apperr.New(apperr.InvalidInput, "unknown archive format")
}

// freeArchiveName returns "name.ext", or "name (n).ext" when taken.
func freeArchiveName(sc *sftp.Client, dir, name, format string) (string, error) {
	for n := 0; n < 1000; n++ {
		candidate := name
		if n > 0 {
			candidate += " (" + strconv.Itoa(n) + ")"
		}
		candidate += "." + format
		_, err := sc.Lstat(path.Join(dir, candidate))
		if errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
		if err != nil {
			return "", mapErr("compress", err)
		}
	}
	return "", apperr.New(apperr.FSExists, "no free archive name left")
}

// Compress runs command, which must be the one PlanCompress returned for
// the same path and format. It fails with FS_STALE when the plan no
// longer holds, for instance when the archive name was taken meanwhile.
func (s *Service) Compress(ctx context.Context, serverID, p, format, command string) (CompressPlan, error) {
	plan, err := s.PlanCompress(ctx, serverID, p, format)
	if err != nil {
		return CompressPlan{}, err
	}
	if plan.Command != command {
		return CompressPlan{}, apperr.New(apperr.FSStale, "the archive plan changed")
	}
	client, err := s.conns.Client(serverID)
	if err != nil {
		return CompressPlan{}, err
	}
	// TODO(M7): record the command in audit.
	s.log.Info("compress", "server", serverID, "app", "files", "op", "compress")
	if err := run(ctx, client, command); err != nil {
		// Do not leave a half-written archive behind.
		if sc, serr := s.conns.SFTP(serverID); serr == nil {
			sc.Remove(plan.Output) //nolint:errcheck,gosec // best effort cleanup
		}
		return CompressPlan{}, err
	}
	return plan, nil
}
