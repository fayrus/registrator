package etcdtls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempCert(t *testing.T) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	cf, err := os.Create(certFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatal(err)
	}
	if err := cf.Close(); err != nil {
		t.Fatal(err)
	}

	kf, err := os.Create(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := pem.Encode(kf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		t.Fatal(err)
	}
	if err := kf.Close(); err != nil {
		t.Fatal(err)
	}

	return certFile, keyFile
}

func TestBuild_NoTLS(t *testing.T) {
	cfg, err := Build("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil config when no cert files provided")
	}
}

func TestBuild_KeyPairError(t *testing.T) {
	_, err := Build("/nonexistent/cert.pem", "/nonexistent/key.pem", "")
	if err == nil {
		t.Fatal("expected error for missing cert files, got nil")
	}
}

func TestBuild_WithValidCert(t *testing.T) {
	certFile, keyFile := writeTempCert(t)
	cfg, err := Build(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil TLS config")
	}
}

func TestBuild_CACertError(t *testing.T) {
	certFile, keyFile := writeTempCert(t)
	_, err := Build(certFile, keyFile, "/nonexistent/ca.pem")
	if err == nil {
		t.Fatal("expected error for missing CA cert, got nil")
	}
}

func TestBuild_CACertWithNoValidPEMs(t *testing.T) {
	certFile, keyFile := writeTempCert(t)

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caFile, []byte("not a pem file"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Build(certFile, keyFile, caFile)
	if err == nil {
		t.Fatal("expected error for CA file with no valid PEM certificates, got nil")
	}
}

func TestBuild_WithValidCACert(t *testing.T) {
	certFile, keyFile := writeTempCert(t)
	cfg, err := Build(certFile, keyFile, certFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || cfg.RootCAs == nil {
		t.Fatal("expected TLS config with RootCAs set")
	}
}
