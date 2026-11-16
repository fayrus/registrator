package etcdtls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// Build returns a TLS config from the given cert/key/CA file paths.
// Returns nil when all three are empty (no TLS). A CA file alone enables
// server-authenticated TLS without a client certificate; certFile and keyFile
// must be set together.
func Build(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" && keyFile == "" && caFile == "" {
		return nil, nil
	}
	if (certFile == "") != (keyFile == "") {
		return nil, errors.New("ETCD_CERT_FILE and ETCD_KEY_FILE must be set together")
	}

	cfg := &tls.Config{}

	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
		}
		cfg.Certificates = []tls.Certificate{cert}
	}

	if caFile != "" {
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("no valid certificates found in CA file: %s", caFile)
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}
