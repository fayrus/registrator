package etcdtls

import (
	"testing"
)

func TestNewClient_NoTLS(t *testing.T) {
	client, err := NewClient("localhost:12379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	_ = client.Close()
}

func TestNewClient_DefaultEndpoint(t *testing.T) {
	client, err := NewClient("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	_ = client.Close()
}

func TestNewClient_ETCDEndpointsEnv(t *testing.T) {
	t.Setenv("ETCD_ENDPOINTS", "localhost:12380, localhost:12381")
	client, err := NewClient("localhost:12379")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	_ = client.Close()
}

func TestNewClient_TLSKeyPairError(t *testing.T) {
	t.Setenv("ETCD_CERT_FILE", "/nonexistent/cert.pem")
	t.Setenv("ETCD_KEY_FILE", "/nonexistent/key.pem")
	_, err := NewClient("localhost:12379")
	if err == nil {
		t.Fatal("expected error for missing TLS files, got nil")
	}
}

func TestBuildEndpoints_UsesHost(t *testing.T) {
	eps := buildEndpoints("myhost:2379")
	if len(eps) != 1 || eps[0] != "myhost:2379" {
		t.Errorf("unexpected endpoints: %v", eps)
	}
}

func TestBuildEndpoints_DefaultWhenEmpty(t *testing.T) {
	eps := buildEndpoints("")
	if len(eps) != 1 || eps[0] != "127.0.0.1:2379" {
		t.Errorf("unexpected endpoints: %v", eps)
	}
}
