package nginx

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"davish/internal/events"
)

func selfSigned(t *testing.T, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "shop.example.com"},
		DNSNames:     []string{"shop.example.com", "example.com"},
		NotBefore:    notAfter.Add(-90 * 24 * time.Hour),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	// A key block first, as in a combined file, then the certificate.
	return append([]byte("-----BEGIN EC PARAMETERS-----\nBggqhkjOPQMBBw==\n-----END EC PARAMETERS-----\n"),
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
}

func TestParseCert(t *testing.T) {
	at := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	c, err := parseCert(selfSigned(t, at))
	if err != nil || !c.NotAfter.Equal(at) || c.DNSNames[1] != "example.com" {
		t.Errorf("cert = %v, %v", c, err)
	}
	if _, err := parseCert([]byte("not a pem")); err == nil {
		t.Error("garbage parsed")
	}
}

type alerts struct{ got []Alert }

func (a *alerts) Emit(name string, p any) {
	if name == events.NotifyNew {
		a.got = append(a.got, p.(Alert))
	}
}

func TestCertWatch(t *testing.T) {
	now := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)
	rec := &alerts{}
	w := newCertWatch(rec)
	w.now = func() time.Time { return now }
	soon := Cert{Path: "/etc/ssl/a.pem", NotAfter: now.Add(10 * 24 * time.Hour).Unix()}

	w.notify("s1", Cert{Path: "/etc/ssl/far.pem", NotAfter: now.Add(60 * 24 * time.Hour).Unix()})
	w.notify("s1", Cert{Path: "/etc/ssl/unreadable.pem", Error: "unreadable"})
	w.notify("s1", soon)
	w.notify("s1", soon) // once is enough
	if len(rec.got) != 1 || rec.got[0].Value != 10 || rec.got[0].Level != "warn" || rec.got[0].ID != "s1:cert:/etc/ssl/a.pem" {
		t.Fatalf("alerts = %+v", rec.got)
	}

	// Renewed, then near expiry again: a new alert.
	soon.NotAfter = now.Add(-2 * 24 * time.Hour).Unix()
	w.notify("s1", soon)
	if len(rec.got) != 2 || rec.got[1].Level != "error" || rec.got[1].Value != -2 {
		t.Errorf("alerts = %+v", rec.got)
	}
}
