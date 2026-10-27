package fs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
)

// tmpName is the sibling a write goes to before it replaces the file.
func tmpName(p string) string {
	return path.Join(path.Dir(p), "."+path.Base(p)+".davish-tmp")
}

// Write replaces the file at p with data atomically: the data goes to a
// temporary sibling that is then renamed over p, so readers never see a
// half-written file. An existing file keeps its mode and owner, and a link
// keeps pointing at the file it pointed to. Asking the user before
// overwriting is the caller's job.
func (s *Service) Write(ctx context.Context, serverID, p string, data []byte) (Entry, error) {
	if err := CheckPath(p); err != nil {
		return Entry{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return Entry{}, err
	}
	target, orig, err := writeTarget(sc, p)
	if err != nil {
		return Entry{}, err
	}

	tmp := tmpName(target)
	if err := writeTemp(ctx, sc, tmp, data, orig); err != nil {
		sc.Remove(tmp) //nolint:errcheck,gosec // best effort cleanup
		return Entry{}, err
	}
	if err := sc.PosixRename(tmp, target); err != nil {
		sc.Remove(tmp) //nolint:errcheck,gosec // best effort cleanup
		return Entry{}, mapErr("write", err)
	}
	return stat(sc, target)
}

// writeTarget resolves links and returns the path to replace and its
// current attributes, nil when the file does not exist yet.
func writeTarget(sc *sftp.Client, p string) (string, *sftp.FileStat, error) {
	target := p
	if fi, err := sc.Lstat(p); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if target, err = sc.RealPath(p); err != nil {
			return "", nil, mapErr("write", err)
		}
	}
	fi, err := sc.Stat(target)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return target, nil, nil
	case err != nil:
		return "", nil, mapErr("write", err)
	case fi.IsDir():
		return "", nil, apperr.New(apperr.InvalidInput, "path is a directory")
	}
	st, ok := fileStat(fi)
	if !ok {
		return "", nil, apperr.New(apperr.Internal, "server sent no file attributes")
	}
	return target, st, nil
}

// writeTemp writes data to tmp and gives it orig's owner and mode.
func writeTemp(ctx context.Context, sc *sftp.Client, tmp string, data []byte, orig *sftp.FileStat) error {
	// A leftover from an interrupted write is ours to replace. O_EXCL then
	// makes sure we never write through a link someone planted there.
	if err := sc.Remove(tmp); err != nil && !errors.Is(err, os.ErrNotExist) {
		return mapErr("write", err)
	}
	f, err := sc.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return mapErr("write", err)
	}
	_, err = io.Copy(f, ctxReader{ctx, bytes.NewReader(data)})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return mapErr("write", err)
	}
	if orig == nil {
		return nil
	}

	// Owner first: chown clears setuid and setgid.
	fi, err := sc.Stat(tmp)
	if err != nil {
		return mapErr("write", err)
	}
	if st, ok := fileStat(fi); !ok || st.UID != orig.UID || st.GID != orig.GID {
		if err := sc.Chown(tmp, int(orig.UID), int(orig.GID)); err != nil {
			// Writing anyway would hand the file to us; refuse instead.
			return apperr.Wrap(apperr.FSPermissionDenied, "cannot keep the file's owner", err)
		}
	}
	if err := sc.Chmod(tmp, fileMode(orig.Mode)); err != nil {
		return mapErr("write", err)
	}
	return nil
}

// fileMode converts POSIX permission bits to what sftp.Chmod expects.
func fileMode(mode uint32) os.FileMode {
	m := os.FileMode(mode & 0o777)
	if mode&sISUID != 0 {
		m |= os.ModeSetuid
	}
	if mode&sISGID != 0 {
		m |= os.ModeSetgid
	}
	if mode&sISVTX != 0 {
		m |= os.ModeSticky
	}
	return m
}

// SaveOptions tunes Save.
type SaveOptions struct {
	// Backup keeps a copy of the current file as <name>.bak-<UTC time>
	// next to it before replacing it.
	Backup bool `json:"backup"`
	// Expect, when set, is the file as the editor read it. If the server
	// copy no longer matches (someone else saved it, or it was deleted),
	// Save fails with FS_CHANGED and writes nothing.
	Expect *Expect `json:"expect,omitempty"`
}

// Expect identifies a version of a file. SFTP times have one-second
// precision, so the size is compared too.
type Expect struct {
	ModTime int64 `json:"modTime"`
	Size    int64 `json:"size"`
}

// SaveResult is the saved file, and the backup made, if any.
type SaveResult struct {
	Entry  Entry  `json:"entry"`
	Backup string `json:"backup,omitempty"`
}

// Save writes data to p like Write, first keeping a backup when asked and
// the file exists. The backup never replaces an earlier one.
func (s *Service) Save(ctx context.Context, serverID, p string, data []byte, opts SaveOptions) (SaveResult, error) {
	if err := CheckPath(p); err != nil {
		return SaveResult{}, err
	}
	var res SaveResult
	if opts.Backup || opts.Expect != nil {
		sc, err := s.conns.SFTP(serverID)
		if err != nil {
			return SaveResult{}, err
		}
		target, orig, err := writeTarget(sc, p)
		if err != nil {
			return SaveResult{}, err
		}
		if e := opts.Expect; e != nil && (orig == nil || int64(orig.Mtime) != e.ModTime || int64(orig.Size) != e.Size) { //nolint:gosec // sizes and times fit int64
			return SaveResult{}, apperr.New(apperr.FSChanged, "the file changed on the server since it was opened")
		}
		if opts.Backup && orig != nil {
			if res.Backup, err = backup(ctx, sc, target, time.Now()); err != nil {
				return SaveResult{}, err
			}
		}
	}
	e, err := s.Write(ctx, serverID, p, data)
	if err != nil {
		return SaveResult{}, err
	}
	res.Entry = e
	return res, nil
}

// backupName is "<name>.bak-20261027T081000Z" next to p.
func backupName(p string, at time.Time) string {
	return p + ".bak-" + at.UTC().Format("20060102T150405Z")
}

// backup copies p to its backup name, keeping mode and times, and returns
// the backup's path. Two saves in the same second get "-2", "-3"…
func backup(ctx context.Context, sc *sftp.Client, p string, at time.Time) (string, error) {
	src, err := sc.Open(p)
	if err != nil {
		return "", mapErr("backup", err)
	}
	defer src.Close() //nolint:errcheck // read only
	fi, err := src.Stat()
	if err != nil {
		return "", mapErr("backup", err)
	}

	base := backupName(p, at)
	var dst *sftp.File
	name := base
	for n := 2; ; n++ {
		dst, err = sc.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if err == nil {
			break
		}
		if _, serr := sc.Lstat(name); serr != nil || n > 100 {
			return "", mapErr("backup", err) // not a name clash: permission and the like
		}
		name = base + "-" + strconv.Itoa(n)
	}
	_, err = io.Copy(dst, ctxReader{ctx, src})
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = sc.Chmod(name, fi.Mode().Perm())
	}
	if err == nil {
		err = sc.Chtimes(name, fi.ModTime(), fi.ModTime())
	}
	if err != nil {
		sc.Remove(name) //nolint:errcheck,gosec // best effort cleanup
		return "", mapErr("backup", err)
	}
	return name, nil
}
