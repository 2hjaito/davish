package transfer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"

	"davish/internal/apperr"
)

// partName is where an item is written before it gets its final name, so
// that a canceled or failed transfer never leaves a half-written file
// under the real name.
func partName(name string) string {
	return "." + name + ".davish-part"
}

// countingReader feeds an upload. Size lets sftp.File.ReadFrom send
// packets concurrently, which is several times faster on high latency.
type countingReader struct {
	r    io.Reader
	size int64
	m    *meter
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if merr := c.m.add(n); merr != nil {
		return n, merr
	}
	return n, err
}

func (c *countingReader) Size() int64 { return c.size }

// countingWriter receives a download.
type countingWriter struct {
	w io.Writer
	m *meter
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if merr := c.m.add(n); merr != nil {
		return n, merr
	}
	return n, err
}

// numbered returns name with " (n)" before its extension, keeping double
// extensions like .tar.gz together: "a.txt" → "a (1).txt".
func numbered(name string, n int) string {
	ext := extension(name)
	return fmt.Sprintf("%s (%d)%s", name[:len(name)-len(ext)], n, ext)
}

func extension(name string) string {
	lower := strings.ToLower(name)
	for _, double := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst"} {
		if strings.HasSuffix(lower, double) && len(name) > len(double) {
			return name[len(name)-len(double):]
		}
	}
	ext := path.Ext(name)
	if ext == name { // ".bashrc" is a name, not an extension
		return ""
	}
	return ext
}

// freeName returns name, or the first numbered variant for which exists
// reports false.
func freeName(name string, exists func(string) (bool, error)) (string, error) {
	for n := 0; n < 1000; n++ {
		candidate := name
		if n > 0 {
			candidate = numbered(name, n)
		}
		taken, err := exists(candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", apperr.New(apperr.FSExists, "no free name left")
}

// safeName rejects names that could step outside the folder they are
// written to. Server names are untrusted when downloading.
func safeName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return apperr.New(apperr.InvalidInput, "unsafe file name")
	}
	return nil
}

// mapErr turns local and SFTP errors into codes the frontend understands.
func mapErr(op string, err error) error {
	var ae *apperr.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ae):
		return ae
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, os.ErrNotExist):
		return apperr.Wrap(apperr.NotFound, op+": no such file or directory", err)
	case errors.Is(err, os.ErrPermission):
		return apperr.Wrap(apperr.FSPermissionDenied, op+": permission denied", err)
	case errors.Is(err, os.ErrExist):
		return apperr.Wrap(apperr.FSExists, op+": already exists", err)
	case errors.Is(err, sftp.ErrSSHFxConnectionLost), errors.Is(err, sftp.ErrSSHFxNoConnection):
		return apperr.Wrap(apperr.ConnNotConnected, op+": connection lost", err)
	}
	return apperr.Wrap(apperr.Internal, op+" failed", err)
}
