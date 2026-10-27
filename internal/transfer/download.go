package transfer

import (
	"context"
	"errors"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/fs"
)

// Download starts one job per remote path, saving into the download
// folder. Existing local files are never replaced: a taken name gets the
// next free "name (n)".
func (s *Service) Download(_ context.Context, serverID string, paths []string) ([]Job, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.opts.DownloadDir, 0o750); err != nil {
		return nil, mapErr("download", err)
	}
	jobs := make([]Job, 0, len(paths))
	for _, p := range paths {
		job, run, err := s.downloadJob(sc, serverID, p)
		if err != nil {
			return jobs, err
		}
		s.start(job, run) //nolint:contextcheck // a job outlives the call that starts it
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Service) downloadJob(sc *sftp.Client, serverID, remote string) (Job, func(context.Context, *meter) error, error) {
	if err := fs.CheckPath(remote); err != nil {
		return Job{}, nil, err
	}
	name := path.Base(remote)
	if err := safeName(name); err != nil {
		return Job{}, nil, err
	}
	// Follow a link given directly: the user wants what it points to.
	fi, err := sc.Stat(remote)
	if err != nil {
		return Job{}, nil, mapErr("download", err)
	}
	if !fi.IsDir() && !fi.Mode().IsRegular() {
		return Job{}, nil, apperr.New(apperr.InvalidInput, "only files and folders can be downloaded")
	}
	dir := s.opts.DownloadDir
	name, err = freeName(name, localExists(dir))
	if err != nil {
		return Job{}, nil, err
	}
	id, err := newID()
	if err != nil {
		return Job{}, nil, err
	}
	job := Job{
		ID: id, Kind: KindDownload, ServerID: serverID, Name: name,
		Src: remote, Dst: filepath.Join(dir, name), Dir: fi.IsDir(),
	}
	run := func(ctx context.Context, m *meter) error {
		if fi.IsDir() {
			return mapErr("download", s.downloadDir(ctx, sc, remote, dir, name, m))
		}
		return mapErr("download", downloadFile(ctx, sc, remote, fi, dir, name, m))
	}
	return job, run, nil
}

func localExists(dir string) func(string) (bool, error) {
	return func(name string) (bool, error) {
		_, err := os.Lstat(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return err == nil, mapErr("download", err)
	}
}

// claim renames part to final unless final appeared in the meantime.
func claim(part, final string) error {
	if _, err := os.Lstat(final); err == nil {
		return apperr.New(apperr.FSExists, "download target already exists")
	}
	return os.Rename(part, final)
}

func downloadFile(ctx context.Context, sc *sftp.Client, remote string, fi os.FileInfo, dir, name string, m *meter) error {
	m.setTotal(fi.Size())
	part := filepath.Join(dir, partName(name))
	// A leftover from an interrupted download is ours to replace.
	if err := os.Remove(part); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	err := getFile(ctx, sc, remote, part, fi.Mode().Perm(), m)
	if err == nil {
		err = claim(part, filepath.Join(dir, name))
	}
	if err != nil {
		os.Remove(part) //nolint:errcheck,gosec // best effort cleanup
	}
	return err
}

// getFile copies remote to a local path that must not exist yet.
func getFile(ctx context.Context, sc *sftp.Client, remote, local string, perm os.FileMode, m *meter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src, err := sc.Open(remote)
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck // read only

	// O_EXCL: never write through a link or over a file already there.
	dst, err := os.OpenFile(local, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm) //nolint:gosec // path built from validated names
	if err != nil {
		return err
	}
	_, err = src.WriteTo(&countingWriter{w: dst, m: m})
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	return err
}

// remoteEntry is one item inside a remote folder being downloaded.
type remoteEntry struct {
	rel  string // slash separated, relative to the folder; every part checked by safeName
	mode iofs.FileMode
	size int64
}

func walkRemote(ctx context.Context, sc *sftp.Client, root string) ([]remoteEntry, int64, error) {
	var entries []remoteEntry
	var total int64
	w := sc.Walk(root)
	for w.Step() {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		if err := w.Err(); err != nil {
			return nil, 0, err
		}
		if w.Path() == root {
			continue
		}
		rel := strings.TrimPrefix(w.Path(), root+"/")
		for _, part := range strings.Split(rel, "/") {
			if err := safeName(part); err != nil {
				return nil, 0, err
			}
		}
		fi := w.Stat() // not following links
		e := remoteEntry{rel: rel, mode: fi.Mode()}
		if fi.Mode().IsRegular() {
			e.size = fi.Size()
			total += e.size
		}
		entries = append(entries, e)
	}
	return entries, total, nil
}

// downloadDir builds the folder under a hidden name and renames it into
// place at the end, so a canceled download leaves nothing half done.
func (s *Service) downloadDir(ctx context.Context, sc *sftp.Client, remote, dir, name string, m *meter) error {
	entries, total, err := walkRemote(ctx, sc, remote)
	if err != nil {
		return err
	}
	m.setTotal(total)

	part := filepath.Join(dir, partName(name))
	if err := os.RemoveAll(part); err != nil {
		return err
	}
	if err := os.Mkdir(part, 0o750); err != nil {
		return err
	}
	final := filepath.Join(dir, name)
	err = s.fillLocalDir(ctx, sc, remote, part, entries, m)
	if err == nil {
		err = claim(part, final)
	}
	if err != nil {
		if rerr := os.RemoveAll(part); rerr != nil {
			s.log.Warn("download cleanup failed", "app", "transfer", "op", "download", "err", rerr)
		}
		return err
	}
	// Modes last: a read-only folder could not have been filled.
	for _, e := range entries {
		if e.mode.IsDir() {
			if err := os.Chmod(filepath.Join(final, filepath.FromSlash(e.rel)), e.mode.Perm()); err != nil {
				s.log.Warn("download chmod failed", "app", "transfer", "op", "download", "err", err)
			}
		}
	}
	return nil
}

func (s *Service) fillLocalDir(ctx context.Context, sc *sftp.Client, remote, part string, entries []remoteEntry, m *meter) error {
	var links []remoteEntry
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		local := filepath.Join(part, filepath.FromSlash(e.rel))
		switch {
		case e.mode.IsDir():
			if err := os.Mkdir(local, 0o750); err != nil {
				return err
			}
		case e.mode.IsRegular():
			if err := getFile(ctx, sc, path.Join(remote, e.rel), local, e.mode.Perm(), m); err != nil {
				return err
			}
		case e.mode&iofs.ModeSymlink != 0:
			links = append(links, e)
		default:
			s.log.Info("download skips special file", "app", "transfer", "op", "download")
		}
	}
	// Links last, so nothing is ever written through one; on a
	// case-insensitive disk a link could otherwise alias a folder. Links
	// that cannot be made (Windows without privilege) are skipped.
	for _, e := range links {
		target, err := sc.ReadLink(path.Join(remote, e.rel))
		if err != nil {
			return err
		}
		if err := os.Symlink(filepath.FromSlash(target), filepath.Join(part, filepath.FromSlash(e.rel))); err != nil {
			s.log.Warn("download skips link", "app", "transfer", "op", "download", "err", err)
		}
	}
	return nil
}
