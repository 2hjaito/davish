package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"davish/internal/apperr"
)

type progress struct{ last Progress }

func (p *progress) Emit(_ string, v any) { p.last = v.(Progress) }

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body)) //nolint:errcheck,gosec
	}
	tw.Close() //nolint:errcheck,gosec
	gz.Close() //nolint:errcheck,gosec
	return buf.Bytes()
}

func zipOf(t *testing.T, files map[string]string, links map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		h := &zip.FileHeader{Name: name}
		h.SetMode(0o755)
		w, _ := zw.CreateHeader(h)
		w.Write([]byte(body)) //nolint:errcheck,gosec
	}
	for name, target := range links {
		h := &zip.FileHeader{Name: name}
		h.SetMode(os.ModeSymlink | 0o777)
		w, _ := zw.CreateHeader(h)
		w.Write([]byte(target)) //nolint:errcheck,gosec
	}
	zw.Close() //nolint:errcheck,gosec
	return buf.Bytes()
}

// fakeGitHub serves one release with archive under assetName, its checksum
// list signed by priv (or a broken signature when tamper).
func fakeGitHub(t *testing.T, version, assetName string, archive []byte, priv ed25519.PrivateKey, tamper bool) *httptest.Server {
	t.Helper()
	sum := sha256.Sum256(archive)
	sums := []byte(hex.EncodeToString(sum[:]) + "  " + assetName + "\n")
	sig := ed25519.Sign(priv, sums)
	if tamper {
		archive = append([]byte{}, archive...)
		archive[len(archive)/2] ^= 0xff
	}
	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/repos/o/r/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck,gosec // test server
			"tag_name": "v" + version, "body": "notes", "html_url": srv.URL + "/page",
			"assets": []map[string]any{
				{"name": assetName, "browser_download_url": srv.URL + "/a/" + assetName, "size": len(archive)},
				{"name": sumsName, "browser_download_url": srv.URL + "/a/" + sumsName},
				{"name": sigName, "browser_download_url": srv.URL + "/a/" + sigName},
			},
		})
	})
	mux.HandleFunc("/a/"+assetName, func(w http.ResponseWriter, _ *http.Request) { w.Write(archive) })                                      //nolint:errcheck,gosec // test server
	mux.HandleFunc("/a/"+sumsName, func(w http.ResponseWriter, _ *http.Request) { w.Write(sums) })                                          //nolint:errcheck,gosec // test server
	mux.HandleFunc("/a/"+sigName, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(base64.StdEncoding.EncodeToString(sig))) }) //nolint:errcheck,gosec // test server
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func service(_ *testing.T, srv *httptest.Server, pub ed25519.PublicKey, goos, exe string) (*Service, *progress) {
	p := &progress{}
	s := New("1.1.0", p, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{
		API: srv.URL, Repo: "o/r", GOOS: goos,
		PublicKey:  base64.StdEncoding.EncodeToString(pub),
		Executable: func() (string, error) { return exe, nil },
	})
	return s, p
}

func code(err error) apperr.Code {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{{"1.10.0", "1.9.2", true}, {"v1.1.0", "1.1.0", false}, {"1.1.1", "1.1.0", true}, {"1.0.9", "1.1.0", false}, {"2.0.0-rc1", "1.9.0", true}} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%s, %s) = %v", c.a, c.b, got)
		}
	}
}

func TestLinuxUpdate(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	name := "Davish-v1.2.0-linux-amd64.tar.gz"
	srv := fakeGitHub(t, "1.2.0", name, tarGz(t, map[string]string{"Davish": "new binary"}), priv, false)
	dir := t.TempDir()
	exe := filepath.Join(dir, "Davish")
	os.WriteFile(exe, []byte("old binary"), 0o755) //nolint:errcheck,gosec
	s, p := service(t, srv, pub, "linux", exe)

	st, err := s.Check(context.Background())
	if err != nil || !st.Available || st.Latest.Version != "1.2.0" || !st.Latest.Signed {
		t.Fatalf("check = %+v, %v", st, err)
	}
	if err := s.Install(context.Background(), "1.2.0"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new binary" { //nolint:gosec // test temp dir
		t.Errorf("app = %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old binary" { //nolint:gosec // test temp dir
		t.Errorf("old = %q", b)
	}
	// Windows has no execute bit; the Linux install is tested there anyway.
	if fi, _ := os.Stat(exe); runtime.GOOS != "windows" && fi.Mode()&0o100 == 0 {
		t.Error("new binary is not executable")
	}
	if p.last.Done == 0 || p.last.Done != p.last.Total || !s.Ready() {
		t.Errorf("progress = %+v, ready = %v", p.last, s.Ready())
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".davish-update-*")); len(left) != 0 {
		t.Errorf("staging left behind: %v", left)
	}
	Cleanup("linux", func() (string, error) { return exe, nil })
	if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
		t.Error("old copy not cleaned up")
	}
}

func TestMacBundleUpdate(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	name := "Davish-v1.2.0-macos-universal.zip"
	archive := zipOf(t,
		map[string]string{"Davish.app/Contents/MacOS/Davish": "new", "Davish.app/Contents/Info.plist": "plist", "__MACOSX/._x": "junk"},
		map[string]string{"Davish.app/Contents/Current": "MacOS"})
	srv := fakeGitHub(t, "1.2.0", name, archive, priv, false)
	apps := t.TempDir()
	exe := filepath.Join(apps, "Davish.app", "Contents", "MacOS", "Davish")
	os.MkdirAll(filepath.Dir(exe), 0o755)   //nolint:errcheck,gosec
	os.WriteFile(exe, []byte("old"), 0o755) //nolint:errcheck,gosec
	s, _ := service(t, srv, pub, "darwin", exe)
	if err := s.Install(context.Background(), "1.2.0"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(exe); string(b) != "new" { //nolint:gosec // test temp dir
		t.Errorf("binary = %q", b)
	}
	if l, _ := os.Readlink(filepath.Join(apps, "Davish.app", "Contents", "Current")); l != "MacOS" {
		t.Errorf("symlink = %q", l)
	}
	if _, err := os.Stat(filepath.Join(apps, "Davish.app.old", "Contents", "MacOS", "Davish")); err != nil {
		t.Error("old bundle not kept")
	}
	if _, err := os.Stat(filepath.Join(apps, "__MACOSX")); err == nil {
		t.Error("metadata unpacked")
	}
}

func TestRefusesTamperedOrForeignSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	name := "Davish-v1.2.0-linux-amd64.tar.gz"
	archive := tarGz(t, map[string]string{"Davish": "new"})
	for label, c := range map[string]struct {
		key    ed25519.PublicKey
		tamper bool
	}{"tampered archive": {pub, true}, "other key": {func() ed25519.PublicKey { p, _, _ := ed25519.GenerateKey(rand.Reader); return p }(), false}} {
		srv := fakeGitHub(t, "1.2.0", name, archive, priv, c.tamper)
		exe := filepath.Join(t.TempDir(), "Davish")
		os.WriteFile(exe, []byte("old"), 0o755) //nolint:errcheck,gosec
		s, _ := service(t, srv, c.key, "linux", exe)
		if err := s.Install(context.Background(), "1.2.0"); code(err) != apperr.UpdateBadSignature {
			t.Errorf("%s: err = %v", label, err)
		}
		if b, _ := os.ReadFile(exe); string(b) != "old" { //nolint:gosec // test temp dir
			t.Errorf("%s: app replaced", label)
		}
	}
}

func TestExtractRefusesEscapes(t *testing.T) {
	for _, archive := range [][]byte{
		zipOf(t, map[string]string{"../evil": "x"}, nil),
		zipOf(t, nil, map[string]string{"Davish.app/l": "../../../etc"}),
	} {
		p := filepath.Join(t.TempDir(), "a.zip")
		os.WriteFile(p, archive, 0o600) //nolint:errcheck,gosec
		if err := extract(p, filepath.Join(t.TempDir(), "out")); err == nil {
			t.Error("escape accepted")
		}
	}
}

func TestOlderReleaseIsNotOffered(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	srv := fakeGitHub(t, "1.0.0", "Davish-v1.0.0-linux-amd64.tar.gz", tarGz(t, map[string]string{"Davish": "x"}), priv, false)
	s, _ := service(t, srv, pub, "linux", "/nowhere/Davish")
	st, err := s.Check(context.Background())
	if err != nil || st.Available {
		t.Fatalf("%+v %v", st, err)
	}
	if err := s.Install(context.Background(), "1.0.0"); err == nil {
		t.Error("installed an older version")
	}
}
