// Package update keeps Davish current from its GitHub releases: it finds a
// newer version, verifies what it downloads against a signed checksum
// list and swaps the running app for the new one (docs/adr/0009-self-update.md).
package update

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"davish/internal/apperr"
	"davish/internal/events"
)

// PublicKey verifies SHA256SUMS.txt.sig; its private half signs releases
// in release.yml (secret DAVISH_UPDATE_KEY, made with cmd/updatesign).
const PublicKey = "pjkeB7e2+bbOttNEP7rtbgNYrx012GcsZ/P/ytcnZSo="

const (
	sumsName = "SHA256SUMS.txt"
	sigName  = sumsName + ".sig"
	// maxDownload bounds an archive; Davish is well under 100 MB.
	maxDownload = 512 << 20
)

// Emitter pushes events to the frontend.
type Emitter interface {
	Emit(name string, payload any)
}

// Options points the service elsewhere for tests.
type Options struct {
	// API is the GitHub API base URL; default https://api.github.com.
	API string
	// Repo is owner/name; default 2hjaito/davish.
	Repo string
	// PublicKey is base64; default PublicKey.
	PublicKey string
	// Executable is the running app's path; default os.Executable.
	Executable func() (string, error)
	// GOOS picks the archive; default runtime.GOOS.
	GOOS   string
	Client *http.Client
}

// Release is a published version.
type Release struct {
	Version string `json:"version"` // without the v
	Notes   string `json:"notes"`
	URL     string `json:"url"` // the release page
	// Signed is set when the release carries a signed checksum list and an
	// archive for this platform, so it can be installed from inside Davish.
	Signed bool  `json:"signed"`
	Size   int64 `json:"size"`

	asset, sums, sig string // download URLs
}

// Status is what a check found.
type Status struct {
	Current   string   `json:"current"`
	Latest    *Release `json:"latest,omitempty"`
	Available bool     `json:"available"`
}

// Progress is the update:progress payload.
type Progress struct {
	Done  int64 `json:"done"`
	Total int64 `json:"total"`
}

// Service checks for and installs updates.
type Service struct {
	current string
	opts    Options
	pub     ed25519.PublicKey
	emit    Emitter
	log     *slog.Logger

	// mu keeps one install at a time and guards ready.
	mu sync.Mutex
	// ready is the path of the swapped-in app, waiting for Restart.
	ready string
}

// New creates the service for the running version (from wails.json).
func New(current string, emit Emitter, log *slog.Logger, opts Options) *Service {
	if opts.API == "" {
		opts.API = "https://api.github.com"
	}
	if opts.Repo == "" {
		opts.Repo = "2hjaito/davish"
	}
	if opts.PublicKey == "" {
		opts.PublicKey = PublicKey
	}
	if opts.Executable == nil {
		opts.Executable = os.Executable
	}
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 5 * time.Minute}
	}
	key, _ := base64.StdEncoding.DecodeString(opts.PublicKey)
	return &Service{current: current, opts: opts, pub: ed25519.PublicKey(key), emit: emit, log: log}
}

// Current is the running version.
func (s *Service) Current() string { return s.current }

// Check asks GitHub for the latest release.
func (s *Service) Check(ctx context.Context) (Status, error) {
	st := Status{Current: s.current}
	rel, err := s.latest(ctx)
	if err != nil {
		return st, err
	}
	st.Latest = rel
	st.Available = Newer(rel.Version, s.current)
	return st, nil
}

func (s *Service) latest(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.opts.API+"/repos/"+s.opts.Repo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Davish/"+s.current)
	resp, err := s.opts.Client.Do(req)
	if err != nil {
		return nil, apperr.Wrap(apperr.UpdateUnavailable, "cannot reach GitHub", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	if resp.StatusCode != http.StatusOK {
		return nil, apperr.New(apperr.UpdateUnavailable, "GitHub answered "+resp.Status)
	}
	var r struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&r); err != nil {
		return nil, apperr.Wrap(apperr.UpdateUnavailable, "unexpected answer from GitHub", err)
	}
	rel := &Release{Version: strings.TrimPrefix(r.TagName, "v"), Notes: r.Body, URL: r.HTMLURL}
	suffix := assetSuffix(s.opts.GOOS)
	for _, a := range r.Assets {
		switch {
		case a.Name == sumsName:
			rel.sums = a.URL
		case a.Name == sigName:
			rel.sig = a.URL
		case strings.HasSuffix(a.Name, suffix) && strings.HasPrefix(a.Name, "Davish-"):
			rel.asset, rel.Size = a.URL, a.Size
		}
	}
	rel.Signed = rel.sums != "" && rel.sig != "" && rel.asset != ""
	return rel, nil
}

// assetSuffix names the archive of a platform, as release.yml packs it.
func assetSuffix(goos string) string {
	switch goos {
	case "darwin":
		return "-macos-universal.zip"
	case "windows":
		return "-windows-amd64.zip"
	}
	return "-linux-amd64.tar.gz"
}

// Newer reports whether version a is above b ("1.10.0" > "1.9.2").
func Newer(a, b string) bool {
	pa, pb := parts(a), parts(b)
	for i := range 3 {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parts(v string) [3]int {
	var out [3]int
	v, _, _ = strings.Cut(strings.TrimPrefix(v, "v"), "-")
	for i, p := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(p)
	}
	return out
}

// Install downloads version, checks it and swaps it in. The running app
// keeps going; Restart starts the new one.
func (s *Service) Install(ctx context.Context, version string) error {
	if !s.mu.TryLock() {
		return apperr.New(apperr.InvalidInput, "an update is already being installed")
	}
	defer s.mu.Unlock()

	rel, err := s.latest(ctx)
	if err != nil {
		return err
	}
	if rel.Version != version || !Newer(rel.Version, s.current) {
		return apperr.New(apperr.InvalidInput, "that version is no longer the latest")
	}
	if !rel.Signed {
		return apperr.New(apperr.UpdateUnsigned, "the release is not signed for updating")
	}
	target, err := s.target()
	if err != nil {
		return err
	}

	sums, err := s.fetch(ctx, rel.sums, 1<<20)
	if err != nil {
		return err
	}
	sig, err := s.fetch(ctx, rel.sig, 4<<10)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sig)))
	if err != nil || len(s.pub) != ed25519.PublicKeySize || !ed25519.Verify(s.pub, sums, raw) {
		s.log.Warn("update signature rejected", "version", rel.Version)
		return apperr.New(apperr.UpdateBadSignature, "checksum list signature does not verify")
	}
	assetName := rel.asset[strings.LastIndex(rel.asset, "/")+1:]
	want := checksum(sums, assetName)
	if want == "" {
		return apperr.New(apperr.UpdateBadSignature, "the archive is not in the signed checksum list")
	}

	// Staging sits next to the app so the final rename stays on one disk.
	stage, err := os.MkdirTemp(filepath.Dir(target), ".davish-update-")
	if err != nil {
		return apperr.Wrap(apperr.UpdateNoWrite, "cannot write next to the app", err)
	}
	defer os.RemoveAll(stage) //nolint:errcheck // best effort

	archive := filepath.Join(stage, assetName)
	if err := s.download(ctx, rel.asset, archive, rel.Size, want); err != nil {
		return err
	}
	unpacked := filepath.Join(stage, "new")
	if err := extract(archive, unpacked); err != nil {
		return apperr.Wrap(apperr.UpdateBadSignature, "cannot unpack the update", err)
	}
	fresh := filepath.Join(unpacked, filepath.Base(target))
	if _, err := os.Stat(fresh); err != nil {
		// The archive names it Davish.app / Davish / Davish.exe whatever the
		// installed copy is called.
		fresh = filepath.Join(unpacked, appName(s.opts.GOOS))
		if _, err := os.Stat(fresh); err != nil {
			return apperr.New(apperr.UpdateBadSignature, "the update does not contain Davish")
		}
	}
	if err := swap(target, fresh); err != nil {
		return apperr.Wrap(apperr.UpdateNoWrite, "cannot replace the app", err)
	}
	s.ready = target
	s.log.Info("update installed", "from", s.current, "to", rel.Version)
	return nil
}

func appName(goos string) string {
	switch goos {
	case "darwin":
		return "Davish.app"
	case "windows":
		return "Davish.exe"
	}
	return "Davish"
}

// target is what gets replaced: the .app bundle on macOS, the binary
// elsewhere.
func (s *Service) target() (string, error) {
	exe, err := s.opts.Executable()
	if err != nil {
		return "", apperr.Wrap(apperr.UpdateNoWrite, "cannot find the running app", err)
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	if s.opts.GOOS == "darwin" {
		app := filepath.Dir(filepath.Dir(filepath.Dir(exe))) // X.app/Contents/MacOS/X
		if !strings.HasSuffix(app, ".app") {
			return "", apperr.New(apperr.UpdateNoWrite, "Davish is not running from an app bundle")
		}
		return app, nil
	}
	return exe, nil
}

func (s *Service) fetch(ctx context.Context, url string, limit int64) ([]byte, error) {
	resp, err := s.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func (s *Service) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Davish/"+s.current)
	resp, err := s.opts.Client.Do(req)
	if err != nil {
		return nil, apperr.Wrap(apperr.UpdateUnavailable, "cannot download the update", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() //nolint:errcheck,gosec // failing
		return nil, apperr.New(apperr.UpdateUnavailable, "download answered "+resp.Status)
	}
	return resp, nil
}

// download writes url to path, reporting progress, and checks its hash.
func (s *Service) download(ctx context.Context, url, path string, size int64, want string) error {
	resp, err := s.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only
	if resp.ContentLength > 0 {
		size = resp.ContentLength
	}
	f, err := os.Create(path) //nolint:gosec // inside our staging dir
	if err != nil {
		return apperr.Wrap(apperr.UpdateNoWrite, "cannot write the download", err)
	}
	defer f.Close() //nolint:errcheck // closed below on success
	h := sha256.New()
	var done int64
	last := time.Time{}
	buf := make([]byte, 64<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if done += int64(n); done > maxDownload {
				return apperr.New(apperr.UpdateBadSignature, "the download is too large")
			}
			h.Write(buf[:n])
			if _, err := f.Write(buf[:n]); err != nil {
				return apperr.Wrap(apperr.UpdateNoWrite, "cannot write the download", err)
			}
			if time.Since(last) > 100*time.Millisecond {
				last = time.Now()
				s.emit.Emit(events.UpdateProgress, Progress{Done: done, Total: size})
			}
		}
		if errors.Is(rerr, io.EOF) {
			break
		}
		if rerr != nil {
			return apperr.Wrap(apperr.UpdateUnavailable, "download interrupted", rerr)
		}
	}
	s.emit.Emit(events.UpdateProgress, Progress{Done: done, Total: done})
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		s.log.Warn("update hash mismatch", "file", filepath.Base(path))
		return apperr.New(apperr.UpdateBadSignature, "the download does not match its checksum")
	}
	return f.Close()
}

// checksum finds name in sha256sum output ("<hex>  <name>" or "<hex> *<name>").
func checksum(sums []byte, name string) string {
	for line := range strings.SplitSeq(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == name && len(f[0]) == 64 {
			return strings.ToLower(f[0])
		}
	}
	return ""
}

// Ready reports whether an installed update waits for a restart.
func (s *Service) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ready != ""
}

// Restart starts the new app; the caller quits right after.
func (s *Service) Restart() error {
	s.mu.Lock()
	target := s.ready
	s.mu.Unlock()
	if target == "" {
		return apperr.New(apperr.InvalidInput, "no update is waiting")
	}
	if err := launch(s.opts.GOOS, target); err != nil {
		return fmt.Errorf("start the new version: %w", err)
	}
	return nil
}

// Relaunch starts a fresh copy of the running app, for settings that only
// apply at start such as the UI language; the caller quits right after.
func (s *Service) Relaunch() error {
	target, err := s.target()
	if err != nil {
		return err
	}
	if err := launch(s.opts.GOOS, target); err != nil {
		return fmt.Errorf("start davish again: %w", err)
	}
	return nil
}

// Cleanup removes what the previous update left: the replaced app.
func Cleanup(goos string, exe func() (string, error)) {
	s := &Service{opts: Options{GOOS: goos, Executable: exe}}
	if target, err := s.target(); err == nil {
		os.RemoveAll(target + ".old") //nolint:errcheck,gosec // best effort
	}
}
