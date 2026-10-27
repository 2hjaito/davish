package nginx

import (
	"crypto/x509"
	"encoding/pem"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"davish/internal/events"
)

// ExpiryWarning is how long before expiry a certificate notifies.
const ExpiryWarning = 14 * 24 * time.Hour

// Cert is one ssl_certificate of an enabled site.
type Cert struct {
	Path     string   `json:"path"`
	NotAfter int64    `json:"notAfter"` // unix seconds; 0 if unreadable
	Names    []string `json:"names"`
	// Error is "unreadable" (often root-only under /etc/letsencrypt) or
	// "invalid".
	Error string `json:"error,omitempty"`
}

// Alert is the notify:new payload of a certificate expiring soon; the
// frontend writes the text from kind, as for sysmon alerts.
type Alert struct {
	ID       string  `json:"id"`
	Level    string  `json:"level"`
	ServerID string  `json:"serverID"`
	Kind     string  `json:"kind"`
	Value    float64 `json:"value"` // days left, negative once expired
	Target   string  `json:"target"`
}

// AlertCertExpiring is the kind of a certificate alert.
const AlertCertExpiring = "cert-expiring"

// certWatch notifies each expiring certificate once per app run, and
// again if it is renewed and then nears expiry again.
type certWatch struct {
	emit Emitter
	now  func() time.Time

	mu sync.Mutex
	// fired maps an alert ID to the expiry it was sent for. Guarded by mu.
	fired map[string]int64
}

func newCertWatch(emit Emitter) *certWatch {
	return &certWatch{emit: emit, now: time.Now, fired: map[string]int64{}}
}

// check reads the certificates of servers and notifies those that expire
// within ExpiryWarning.
func (w *certWatch) check(sc *sftp.Client, serverID string, servers []Server) []Cert {
	seen := map[string]bool{}
	certs := []Cert{}
	for _, s := range servers {
		for _, p := range s.Certs {
			if seen[p] {
				continue
			}
			seen[p] = true
			c := readCert(sc, p)
			certs = append(certs, c)
			w.notify(serverID, c)
		}
	}
	return certs
}

func (w *certWatch) notify(serverID string, c Cert) {
	if c.NotAfter == 0 {
		return
	}
	left := time.Unix(c.NotAfter, 0).Sub(w.now())
	if left > ExpiryWarning {
		return
	}
	id := serverID + ":cert:" + c.Path
	w.mu.Lock()
	if w.fired[id] == c.NotAfter {
		w.mu.Unlock()
		return
	}
	w.fired[id] = c.NotAfter
	w.mu.Unlock()
	level := "warn"
	if left <= 0 {
		level = "error"
	}
	w.emit.Emit(events.NotifyNew, Alert{
		ID: id, Level: level, ServerID: serverID, Kind: AlertCertExpiring,
		Value: float64(int(left.Hours() / 24)), Target: c.Path,
	})
}

func readCert(sc *sftp.Client, p string) Cert {
	c := Cert{Path: p, Names: []string{}}
	text, err := readFile(sc, p)
	if err != nil {
		c.Error = "unreadable"
		return c
	}
	cert, err := parseCert([]byte(text))
	if err != nil {
		c.Error = "invalid"
		return c
	}
	c.NotAfter = cert.NotAfter.Unix()
	c.Names = append(c.Names, cert.DNSNames...)
	if len(c.Names) == 0 && cert.Subject.CommonName != "" {
		c.Names = append(c.Names, cert.Subject.CommonName)
	}
	return c
}

// parseCert reads the first certificate of a PEM file: the site's own,
// ahead of the chain in a fullchain.pem.
func parseCert(data []byte) (*x509.Certificate, error) {
	for {
		block, rest := pem.Decode(data)
		if block == nil {
			return nil, errNoCert
		}
		if block.Type == "CERTIFICATE" {
			return x509.ParseCertificate(block.Bytes)
		}
		data = rest
	}
}

type certError string

func (e certError) Error() string { return string(e) }

const errNoCert = certError("no certificate in file")
