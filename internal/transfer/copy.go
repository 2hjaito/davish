package transfer

import (
	"context"
	"errors"
	"io"
	iofs "io/fs"
	"os"
	"path"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/fs"
)

// copyBuffer is how much a server-to-server copy holds in memory at once.
const copyBuffer = 256 << 10

// Copy starts one job per path, copying it from fromServer into dir on
// toServer. A name already taken in dir gets the next free "name (n)":
// nothing is ever replaced, so nothing needs confirming.
func (s *Service) Copy(_ context.Context, fromServer string, paths []string, toServer, dir string) ([]Job, error) {
	if err := fs.CheckPath(dir); err != nil {
		return nil, err
	}
	src, err := s.conns.SFTP(fromServer)
	if err != nil {
		return nil, err
	}
	dst, err := s.conns.SFTP(toServer)
	if err != nil {
		return nil, err
	}
	if fi, err := dst.Stat(dir); err != nil || !fi.IsDir() {
		return nil, apperr.New(apperr.NotFound, "the destination folder does not exist")
	}
	jobs := make([]Job, 0, len(paths))
	for _, p := range paths {
		job, run, err := s.copyJob(src, dst, fromServer, toServer, p, dir)
		if err != nil {
			return jobs, err
		}
		s.start(job, run) //nolint:contextcheck // a job outlives the call that starts it
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Service) copyJob(src, dst *sftp.Client, fromServer, toServer, p, dir string) (Job, func(context.Context, *meter) error, error) {
	if err := fs.CheckPath(p); err != nil || p == "/" {
		return Job{}, nil, apperr.New(apperr.InvalidInput, "cannot copy this path")
	}
	// Copying a folder into itself on the same server would never end.
	if fromServer == toServer && (dir == p || len(dir) > len(p) && dir[:len(p)+1] == p+"/") {
		return Job{}, nil, apperr.New(apperr.InvalidInput, "cannot copy a folder into itself")
	}
	fi, err := src.Lstat(p)
	if err != nil {
		return Job{}, nil, mapErr("copy", err)
	}
	if !fi.IsDir() && !fi.Mode().IsRegular() {
		return Job{}, nil, apperr.New(apperr.InvalidInput, "only files and folders can be copied")
	}
	name := path.Base(p)
	if err := safeName(name); err != nil {
		return Job{}, nil, err
	}
	name, err = freeName(name, remoteExists(dst, dir))
	if err != nil {
		return Job{}, nil, err
	}
	id, err := newID()
	if err != nil {
		return Job{}, nil, err
	}
	job := Job{
		ID: id, Kind: KindCopy, ServerID: toServer, FromServerID: fromServer,
		Name: name, Src: p, Dst: path.Join(dir, name), Dir: fi.IsDir(),
	}
	run := func(ctx context.Context, m *meter) error {
		if fi.IsDir() {
			return mapErr("copy", s.copyDir(ctx, src, dst, p, dir, name, m))
		}
		m.setTotal(fi.Size())
		return mapErr("copy", copyFileInto(ctx, src, dst, p, fi, dir, name, m))
	}
	return job, run, nil
}

// copyFileInto writes the file next to its target and renames it into
// place, refusing to overwrite.
func copyFileInto(ctx context.Context, src, dst *sftp.Client, from string, fi os.FileInfo, dir, name string, m *meter) error {
	tmp := path.Join(dir, partName(name))
	// A leftover from an interrupted copy is ours to replace.
	if err := dst.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err := copyFile(ctx, src, dst, from, tmp, fi.Size(), fi.Mode().Perm(), m)
	if err == nil {
		err = dst.Rename(tmp, path.Join(dir, name)) // refuses to overwrite
	}
	if err != nil {
		dst.Remove(tmp) //nolint:errcheck,gosec // best effort cleanup
	}
	return err
}

// copyFile streams one file between servers to a path that must not exist.
func copyFile(ctx context.Context, src, dst *sftp.Client, from, to string, size int64, mode os.FileMode, m *meter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	in, err := src.Open(from)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck // read only
	// O_EXCL: never write through a link someone planted at this name.
	out, err := dst.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	_, err = io.CopyBuffer(out, &countingReader{r: in, size: size, m: m}, make([]byte, copyBuffer))
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return dst.Chmod(to, mode)
}

// copyDir builds the folder under a hidden name on the destination and
// renames it into place at the end, so a canceled copy leaves nothing.
func (s *Service) copyDir(ctx context.Context, src, dst *sftp.Client, from, dir, name string, m *meter) error {
	entries, total, err := walkRemote(ctx, src, from)
	if err != nil {
		return err
	}
	m.setTotal(total)

	tmp := path.Join(dir, partName(name))
	if _, err := dst.Lstat(tmp); err == nil {
		if err := dst.RemoveAll(tmp); err != nil {
			return err
		}
	}
	if err := dst.Mkdir(tmp); err != nil {
		return err
	}
	err = s.fillCopy(ctx, src, dst, from, tmp, entries, m)
	if err == nil {
		err = dst.Rename(tmp, path.Join(dir, name)) // refuses to overwrite
	}
	if err != nil {
		if rerr := dst.RemoveAll(tmp); rerr != nil {
			s.log.Warn("copy cleanup failed", "app", "transfer", "op", "copy", "err", rerr)
		}
		return err
	}
	// Modes last: a read-only folder could not have been filled.
	final := path.Join(dir, name)
	setMode := func(p string, mode iofs.FileMode) {
		if err := dst.Chmod(p, mode.Perm()); err != nil {
			s.log.Warn("copy chmod failed", "app", "transfer", "op", "copy", "err", err)
		}
	}
	if fi, err := src.Stat(from); err == nil {
		setMode(final, fi.Mode())
	}
	for _, e := range entries {
		if e.mode.IsDir() {
			setMode(path.Join(final, e.rel), e.mode)
		}
	}
	return nil
}

func (s *Service) fillCopy(ctx context.Context, src, dst *sftp.Client, from, tmp string, entries []remoteEntry, m *meter) error {
	var links []remoteEntry
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		to := path.Join(tmp, e.rel)
		switch {
		case e.mode.IsDir():
			if err := dst.Mkdir(to); err != nil {
				return err
			}
		case e.mode.IsRegular():
			if err := copyFile(ctx, src, dst, path.Join(from, e.rel), to, e.size, e.mode.Perm(), m); err != nil {
				return err
			}
		case e.mode&iofs.ModeSymlink != 0:
			links = append(links, e)
		default:
			s.log.Info("copy skips special file", "app", "transfer", "op", "copy")
		}
	}
	// Links last, so nothing is ever written through one.
	for _, e := range links {
		target, err := src.ReadLink(path.Join(from, e.rel))
		if err != nil {
			return err
		}
		if err := dst.Symlink(target, path.Join(tmp, e.rel)); err != nil {
			return err
		}
	}
	return nil
}
