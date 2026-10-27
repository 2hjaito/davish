package fs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"

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
