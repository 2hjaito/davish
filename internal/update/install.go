package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// extract unpacks a .zip or .tar.gz into dir. Entries may not leave dir,
// and symlinks (frameworks in a .app) may only point inside it.
func extract(archive, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // an app is world-readable
		return err
	}
	if strings.HasSuffix(archive, ".zip") {
		return unzip(archive, dir)
	}
	return untar(archive, dir)
}

// inside resolves name under dir, refusing ../ and absolute paths.
func inside(dir, name string) (string, error) {
	p := filepath.Join(dir, filepath.FromSlash(name))
	if p != dir && !strings.HasPrefix(p, dir+string(filepath.Separator)) {
		return "", fmt.Errorf("%q leaves the archive", name)
	}
	return p, nil
}

// skip drops macOS metadata ditto may add.
func skip(name string) bool {
	return strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(filepath.Base(name), "._")
}

func link(dir, path, target string) error {
	if filepath.IsAbs(target) {
		return fmt.Errorf("absolute symlink %q", target)
	}
	rel, err := filepath.Rel(dir, filepath.Dir(path))
	if err != nil {
		return err
	}
	if _, err := inside(dir, filepath.ToSlash(filepath.Join(rel, target))); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // an app is world-readable
		return err
	}
	return os.Symlink(target, path)
}

func write(path string, mode os.FileMode, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:gosec // an app is world-readable
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm()|0o600) //nolint:gosec // inside staging
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, io.LimitReader(r, maxDownload)); err != nil {
		f.Close() //nolint:errcheck,gosec // failing
		return err
	}
	return f.Close()
}

func unzip(archive, dir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer zr.Close() //nolint:errcheck // read-only
	for _, f := range zr.File {
		if skip(f.Name) {
			continue
		}
		path, err := inside(dir, f.Name)
		if err != nil {
			return err
		}
		mode := f.Mode()
		switch {
		case mode.IsDir():
			err = os.MkdirAll(path, 0o755) //nolint:gosec // an app is world-readable
		case mode&os.ModeSymlink != 0:
			var target []byte
			if target, err = readZip(f, 4096); err == nil {
				err = link(dir, path, string(target))
			}
		default:
			var rc io.ReadCloser
			if rc, err = f.Open(); err == nil {
				err = write(path, mode, rc)
				rc.Close() //nolint:errcheck,gosec // read-only
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func readZip(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close() //nolint:errcheck // read-only
	return io.ReadAll(io.LimitReader(rc, limit))
}

func untar(archive, dir string) error {
	f, err := os.Open(archive) //nolint:gosec // our own download
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // read-only
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if skip(h.Name) {
			continue
		}
		path, err := inside(dir, h.Name)
		if err != nil {
			return err
		}
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, 0o755) //nolint:gosec // an app is world-readable
		case tar.TypeSymlink:
			err = link(dir, path, h.Linkname)
		case tar.TypeReg:
			err = write(path, os.FileMode(h.Mode), tr) //nolint:gosec // mode bits only
		default:
			continue // devices, hard links: not in a Davish release
		}
		if err != nil {
			return err
		}
	}
}

// swap puts fresh where target is, keeping the old one as target.old until
// the next start. Renaming works while the app runs, on every OS.
func swap(target, fresh string) error {
	old := target + ".old"
	if err := os.RemoveAll(old); err != nil {
		return err
	}
	if err := os.Rename(target, old); err != nil {
		return err
	}
	if err := os.Rename(fresh, target); err != nil {
		os.Rename(old, target) //nolint:errcheck,gosec // put the old one back
		return err
	}
	return nil
}

// launch starts the new app without waiting for it.
func launch(goos, target string) error {
	var cmd *exec.Cmd
	if goos == "darwin" {
		// -n: a new instance while this one is still quitting.
		cmd = exec.Command("open", "-n", target) //nolint:gosec // our own app path
	} else {
		cmd = exec.Command(target) //nolint:gosec // our own app path
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
