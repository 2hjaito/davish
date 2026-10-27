// Package fs implements SFTP file operations, including atomic writes.
package fs

import (
	"context"
	"errors"
	"io"
	iofs "io/fs"
	"log/slog"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"davish/internal/apperr"
)

// Conns hands out a server's live SSH client and its shared SFTP client.
type Conns interface {
	Client(serverID string) (*ssh.Client, error)
	SFTP(serverID string) (*sftp.Client, error)
}

// MaxRead is the largest file Read returns; bigger files go through a
// download instead of being held in the webview.
const MaxRead = 10 << 20

// Entry kinds.
const (
	KindFile  = "file"
	KindDir   = "dir"
	KindLink  = "link"
	KindOther = "other" // device, socket, FIFO
)

// Entry describes one file or directory.
type Entry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Kind string `json:"kind"`
	// Dir is true for directories and for links that point to one, so the
	// UI can open both the same way.
	Dir bool `json:"dir"`
	// Target is where a link points, as stored; empty for other kinds.
	Target  string `json:"target,omitempty"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"` // like ls -l: "drwxr-xr-x"
	Perm    uint32 `json:"perm"` // permission bits, 0o7777 at most
	ModTime int64  `json:"modTime"`
	UID     uint32 `json:"uid"`
	GID     uint32 `json:"gid"`
}

// File is a file's content with the stat it was read with, so a later write
// can tell whether the file changed in between.
type File struct {
	Entry
	Data []byte `json:"data"`
}

// Service runs file operations over each server's SFTP client.
type Service struct {
	conns Conns
	log   *slog.Logger
}

// New creates the service.
func New(conns Conns, log *slog.Logger) *Service {
	return &Service{conns: conns, log: log}
}

// Home returns the directory the SFTP session starts in, normally the
// user's home.
func (s *Service) Home(_ context.Context, serverID string) (string, error) {
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return "", err
	}
	dir, err := sc.Getwd()
	if err != nil {
		return "", mapErr("home", err)
	}
	return dir, nil
}

// linkWorkers bounds the stat calls in flight when List resolves links.
const linkWorkers = 16

// List returns the entries of dir sorted by name. Links are resolved so
// that links to directories can be opened like one.
func (s *Service) List(ctx context.Context, serverID, dir string) ([]Entry, error) {
	if err := CheckPath(dir); err != nil {
		return nil, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return nil, err
	}
	infos, err := sc.ReadDir(dir)
	if err != nil {
		return nil, mapErr("list", err)
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Name() < infos[j].Name() })

	out := make([]Entry, len(infos))
	var links []int
	for i, fi := range infos {
		out[i] = toEntry(path.Join(dir, fi.Name()), fi)
		if out[i].Kind == KindLink {
			links = append(links, i)
		}
	}
	resolveLinks(ctx, sc, out, links)
	return out, ctx.Err()
}

// resolveLinks fills Target and Dir of the entries at idx. Broken links
// keep Dir false.
func resolveLinks(ctx context.Context, sc *sftp.Client, out []Entry, idx []int) {
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range min(linkWorkers, len(idx)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				e := &out[i]
				e.Target, _ = sc.ReadLink(e.Path)
				if fi, err := sc.Stat(e.Path); err == nil {
					e.Dir = fi.IsDir()
				}
			}
		}()
	}
	for _, i := range idx {
		if ctx.Err() != nil {
			break
		}
		jobs <- i
	}
	close(jobs)
	wg.Wait()
}

// Stat describes p without following a final link.
func (s *Service) Stat(_ context.Context, serverID, p string) (Entry, error) {
	if err := CheckPath(p); err != nil {
		return Entry{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return Entry{}, err
	}
	return stat(sc, p)
}

func stat(sc *sftp.Client, p string) (Entry, error) {
	fi, err := sc.Lstat(p)
	if err != nil {
		return Entry{}, mapErr("stat", err)
	}
	e := toEntry(p, fi)
	if e.Kind == KindLink {
		e.Target, _ = sc.ReadLink(p)
		if ti, err := sc.Stat(p); err == nil {
			e.Dir = ti.IsDir()
		}
	}
	return e, nil
}

// Read returns the content of the file at p, following links. Files over
// MaxRead fail with FS_TOO_LARGE.
func (s *Service) Read(ctx context.Context, serverID, p string) (File, error) {
	if err := CheckPath(p); err != nil {
		return File{}, err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return File{}, err
	}
	f, err := sc.Open(p)
	if err != nil {
		return File{}, mapErr("read", err)
	}
	defer f.Close() //nolint:errcheck // read only
	fi, err := f.Stat()
	if err != nil {
		return File{}, mapErr("read", err)
	}
	if fi.IsDir() {
		return File{}, apperr.New(apperr.InvalidInput, "path is a directory")
	}
	if fi.Size() > MaxRead {
		return File{}, apperr.New(apperr.FSTooLarge, "file is too large to open")
	}
	// The size can grow while reading; never hold more than MaxRead.
	data, err := io.ReadAll(io.LimitReader(ctxReader{ctx, f}, MaxRead+1))
	if err != nil {
		return File{}, mapErr("read", err)
	}
	if len(data) > MaxRead {
		return File{}, apperr.New(apperr.FSTooLarge, "file is too large to open")
	}
	return File{Entry: toEntry(p, fi), Data: data}, nil
}

// Mkdir creates the directory p; its parent must exist.
func (s *Service) Mkdir(_ context.Context, serverID, p string) error {
	if err := CheckPath(p); err != nil {
		return err
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return err
	}
	if err := mustNotExist(sc, p); err != nil {
		return err
	}
	if err := sc.Mkdir(p); err != nil {
		return mapErr("mkdir", err)
	}
	return nil
}

// Rename moves from to to. It never replaces an existing to.
func (s *Service) Rename(_ context.Context, serverID, from, to string) error {
	if err := CheckPath(from); err != nil {
		return err
	}
	if err := CheckPath(to); err != nil {
		return err
	}
	if from == "/" || from == to {
		return apperr.New(apperr.InvalidInput, "cannot rename this path")
	}
	sc, err := s.conns.SFTP(serverID)
	if err != nil {
		return err
	}
	if err := mustNotExist(sc, to); err != nil {
		return err
	}
	// Plain SFTP rename refuses to overwrite, unlike posix-rename.
	if err := sc.Rename(from, to); err != nil {
		return mapErr("rename", err)
	}
	return nil
}

func mustNotExist(sc *sftp.Client, p string) error {
	_, err := sc.Lstat(p)
	switch {
	case err == nil:
		return apperr.New(apperr.FSExists, "path already exists")
	case errors.Is(err, os.ErrNotExist):
		return nil
	}
	return mapErr("stat", err)
}

// CheckPath accepts only clean absolute paths, so that what the user
// confirmed is exactly what gets touched.
func CheckPath(p string) error {
	if !path.IsAbs(p) || path.Clean(p) != p || strings.ContainsRune(p, 0) {
		return apperr.New(apperr.InvalidInput, "path must be absolute and clean")
	}
	return nil
}

// mapErr turns SFTP errors into codes the frontend understands. The cause
// is kept for logging only.
func mapErr(op string, err error) error {
	var ae *apperr.Error
	switch {
	case errors.As(err, &ae):
		return ae
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, os.ErrNotExist):
		return apperr.Wrap(apperr.NotFound, op+": no such file or directory", err)
	case errors.Is(err, os.ErrPermission):
		return apperr.Wrap(apperr.FSPermissionDenied, op+": permission denied", err)
	case errors.Is(err, sftp.ErrSSHFxConnectionLost), errors.Is(err, sftp.ErrSSHFxNoConnection):
		return apperr.Wrap(apperr.ConnNotConnected, op+": connection lost", err)
	}
	return apperr.Wrap(apperr.Internal, op+" failed", err)
}

// ctxReader stops a long read when ctx is canceled.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// fileStat returns the raw attributes when the server sent them.
func fileStat(fi iofs.FileInfo) (*sftp.FileStat, bool) {
	st, ok := fi.Sys().(*sftp.FileStat)
	return st, ok && st != nil
}
