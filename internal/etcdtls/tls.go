package etcdtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// Build returns a TLS config from the given cert/key/CA file paths.
// Returns nil if certFile or keyFile are empty (no TLS).
func Build(certFile, keyFile, caFile string) (*tls.Config, error) {
	if certFile == "" || keyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}

	if caFile != "" {
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		cfg.RootCAs = pool
	}

	return cfg, nil
}
