package transfer

import (
	"context"
	"errors"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
	"davish/internal/fs"
	"davish/internal/shellx"
)

// UploadItem is a local file or folder about to be uploaded, as the UI
// shows it before starting.
type UploadItem struct {
	Local string `json:"local"`
	Name  string `json:"name"`
	Dir   bool   `json:"dir"`
	Size  int64  `json:"size"` // 0 for folders until they are walked
	// Exists is true when the target folder already has Name.
	Exists    bool `json:"exists"`
	ExistsDir bool `json:"existsDir"`
	// ReplaceCommand is the operation replacing the existing file runs,
	// for the confirmation dialog. It is empty when replacing is not
	// offered: folders are never merged or replaced, only kept side by side.
	ReplaceCommand string `json:"replaceCommand,omitempty"`
}

// UploadRequest starts one upload. Without Replace an existing name is
// kept and the upload gets the next free "name (n)".
type UploadRequest struct {
	Local   string `json:"local"`
	Replace bool   `json:"replace"`
}

// PlanUpload describes what uploading locals into dir would do, so the
// UI can ask about names that already exist.
func (s *Service) PlanUpload(_ context.Context, serverID, dir string, locals []string) ([]UploadItem, error) {
	if err := fs.CheckPath(dir); err != nil {
		return nil, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	items := make([]UploadItem, 0, len(locals))
	for _, local := range locals {
		fi, err := statLocal(local)
		if err != nil {
			return nil, err
		}
		it := UploadItem{Local: local, Name: fi.Name(), Dir: fi.IsDir()}
		if !it.Dir {
			it.Size = fi.Size()
		}
		dst := path.Join(dir, it.Name)
		rfi, err := sc.Lstat(dst)
		switch {
		case err == nil:
			it.Exists, it.ExistsDir = true, rfi.IsDir()
			if !it.Dir && rfi.Mode().IsRegular() {
				it.ReplaceCommand = shellx.Cmd("put", local, dst).String()
			}
		case !errors.Is(err, os.ErrNotExist):
			return nil, mapErr("upload", err)
		}
		items = append(items, it)
	}
	return items, nil
}

// statLocal follows a link given directly, since the user picked what it
// points to, and accepts only files and folders.
func statLocal(local string) (os.FileInfo, error) {
	if !filepath.IsAbs(local) {
		return nil, apperr.New(apperr.InvalidInput, "local path must be absolute")
	}
	fi, err := os.Stat(local)
	if err != nil {
		return nil, mapErr("upload", err)
	}
	if !fi.IsDir() && !fi.Mode().IsRegular() {
		return nil, apperr.New(apperr.InvalidInput, "only files and folders can be uploaded")
	}
	return fi, nil
}

// Upload starts one job per request, uploading into dir.
func (s *Service) Upload(_ context.Context, serverID, dir string, reqs []UploadRequest) ([]Job, error) {
	if err := fs.CheckPath(dir); err != nil {
		return nil, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(reqs))
	for _, req := range reqs {
		job, run, err := s.uploadJob(sc, serverID, dir, req)
		if err != nil {
			return jobs, err
		}
		s.start(job, run) //nolint:contextcheck // a job outlives the call that starts it
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Service) uploadJob(sc *sftp.Client, serverID, dir string, req UploadRequest) (Job, func(context.Context, *meter) error, error) {
	fi, err := statLocal(req.Local)
	if err != nil {
		return Job{}, nil, err
	}
	name := fi.Name()
	var replaceMode *os.FileMode
	if req.Replace {
		rfi, err := sc.Lstat(path.Join(dir, name))
		if err != nil || fi.IsDir() || !rfi.Mode().IsRegular() {
			return Job{}, nil, apperr.New(apperr.InvalidInput, "only an existing file can be replaced")
		}
		mode := rfi.Mode().Perm()
		replaceMode = &mode
	} else {
		name, err = freeName(name, remoteExists(sc, dir))
		if err != nil {
			return Job{}, nil, err
		}
	}
	id, err := newID()
	if err != nil {
		return Job{}, nil, err
	}
	job := Job{
		ID: id, Kind: KindUpload, ServerID: serverID, Name: name,
		Src: req.Local, Dst: path.Join(dir, name), Dir: fi.IsDir(),
	}
	run := func(ctx context.Context, m *meter) error {
		if fi.IsDir() {
			return mapErr("upload", s.uploadDir(ctx, sc, req.Local, dir, name, m))
		}
		return mapErr("upload", uploadFile(ctx, sc, req.Local, fi, dir, name, replaceMode, m))
	}
	return job, run, nil
}

func remoteExists(sc *sftp.Client, dir string) func(string) (bool, error) {
	return func(name string) (bool, error) {
		_, err := sc.Lstat(path.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return err == nil, mapErr("upload", err)
	}
}

// uploadFile writes the file next to its target and renames it into
// place. replaceMode is the mode of the file it replaces, nil when the
// name is new; a new name is never allowed to replace anything.
func uploadFile(ctx context.Context, sc *sftp.Client, local string, fi os.FileInfo, dir, name string, replaceMode *os.FileMode, m *meter) error {
	m.setTotal(fi.Size())
	tmp := path.Join(dir, partName(name))
	dst := path.Join(dir, name)
	// A leftover from an interrupted upload is ours to replace.
	if err := sc.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	mode := fi.Mode().Perm()
	if replaceMode != nil {
		mode = *replaceMode
	}
	err := putFile(ctx, sc, local, tmp, fi.Size(), mode, m)
	if err == nil {
		if replaceMode != nil {
			err = sc.PosixRename(tmp, dst)
		} else {
			err = sc.Rename(tmp, dst) // refuses to overwrite
		}
	}
	if err != nil {
		sc.Remove(tmp) //nolint:errcheck,gosec // best effort cleanup
	}
	return err
}

// putFile copies local to a remote path that must not exist yet.
func putFile(ctx context.Context, sc *sftp.Client, local, remote string, size int64, mode os.FileMode, m *meter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src, err := os.Open(local) //nolint:gosec // the user picked this file
	if err != nil {
		return err
	}
	defer src.Close() //nolint:errcheck // read only

	// O_EXCL: never write through a link someone planted at this name.
	dst, err := sc.OpenFile(remote, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	_, err = dst.ReadFrom(&countingReader{r: src, size: size, m: m})
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return sc.Chmod(remote, mode)
}

// localEntry is one item inside a local folder being uploaded.
type localEntry struct {
	rel  string // slash separated, relative to the folder
	mode iofs.FileMode
	size int64
}

func walkLocal(root string) ([]localEntry, int64, error) {
	var entries []localEntry
	var total int64
	err := filepath.WalkDir(root, func(p string, d iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		e := localEntry{rel: filepath.ToSlash(rel), mode: info.Mode()}
		if info.Mode().IsRegular() {
			e.size = info.Size()
			total += e.size
		}
		entries = append(entries, e)
		return nil
	})
	return entries, total, err
}

// uploadDir builds the folder under a hidden name and renames it into
// place at the end, so a canceled upload leaves nothing half done.
func (s *Service) uploadDir(ctx context.Context, sc *sftp.Client, local, dir, name string, m *meter) error {
	entries, total, err := walkLocal(local)
	if err != nil {
		return err
	}
	m.setTotal(total)

	tmp := path.Join(dir, partName(name))
	if _, err := sc.Lstat(tmp); err == nil {
		if err := sc.RemoveAll(tmp); err != nil {
			return err
		}
	}
	if err := sc.Mkdir(tmp); err != nil {
		return err
	}
	err = s.fillRemoteDir(ctx, sc, local, tmp, entries, m)
	if err == nil {
		err = sc.Rename(tmp, path.Join(dir, name)) // refuses to overwrite
	}
	if err != nil {
		if rerr := sc.RemoveAll(tmp); rerr != nil {
			s.log.Warn("upload cleanup failed", "app", "transfer", "op", "upload", "err", rerr)
		}
		return err
	}
	// Modes last: a read-only folder could not have been filled.
	dst := path.Join(dir, name)
	setMode := func(p string, mode iofs.FileMode) {
		if err := sc.Chmod(p, mode.Perm()); err != nil {
			s.log.Warn("upload chmod failed", "app", "transfer", "op", "upload", "err", err)
		}
	}
	if fi, err := os.Stat(local); err == nil {
		setMode(dst, fi.Mode())
	}
	for _, e := range entries {
		if e.mode.IsDir() {
			setMode(path.Join(dst, e.rel), e.mode)
		}
	}
	return nil
}

func (s *Service) fillRemoteDir(ctx context.Context, sc *sftp.Client, local, tmp string, entries []localEntry, m *meter) error {
	var links []localEntry
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		remote := path.Join(tmp, e.rel)
		switch {
		case e.mode.IsDir():
			if err := sc.Mkdir(remote); err != nil {
				return err
			}
		case e.mode.IsRegular():
			src := filepath.Join(local, filepath.FromSlash(e.rel))
			if err := putFile(ctx, sc, src, remote, e.size, e.mode.Perm(), m); err != nil {
				return err
			}
		case e.mode&iofs.ModeSymlink != 0:
			links = append(links, e)
		default:
			s.log.Info("upload skips special file", "app", "transfer", "op", "upload")
		}
	}
	// Links last, so nothing is ever written through one.
	for _, e := range links {
		target, err := os.Readlink(filepath.Join(local, filepath.FromSlash(e.rel)))
		if err != nil {
			return err
		}
		if err := sc.Symlink(filepath.ToSlash(target), path.Join(tmp, e.rel)); err != nil {
			return err
		}
	}
	return nil
}
