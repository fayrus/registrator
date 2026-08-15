package kvutil

import (
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
)

type serviceFromKVCase struct {
	name     string
	key      string
	value    string
	want     bool
	wantName string
	wantID   string
	wantIP   string
	wantPort int
}

func TestServiceFromKV(t *testing.T) {
	tests := []serviceFromKVCase{
		{
			name:     "valid IPv4 service",
			key:      "/services/web/host:web:80",
			value:    "10.0.0.1:8080",
			want:     true,
			wantName: "web",
			wantID:   "host:web:80",
			wantIP:   "10.0.0.1",
			wantPort: 8080,
		},
		{
			name:     "valid IPv6 service",
			key:      "/services/api/host:api:9000",
			value:    "[2001:db8::1]:9000",
			want:     true,
			wantName: "api",
			wantID:   "host:api:9000",
			wantIP:   "2001:db8::1",
			wantPort: 9000,
		},
		{
			name:  "wrong prefix",
			key:   "/other/web/host:web:80",
			value: "10.0.0.1:8080",
		},
		{
			name:  "missing service ID",
			key:   "/services/web",
			value: "10.0.0.1:8080",
		},
		{
			name:  "invalid address",
			key:   "/services/web/host:web:80",
			value: "not-an-address",
		},
		{
			name:  "invalid port",
			key:   "/services/web/host:web:80",
			value: "10.0.0.1:not-a-port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, ok := ServiceFromKV("/services/", tt.key, tt.value)
			assertServiceFromKV(t, tt, service, ok)
		})
	}
}

func assertServiceFromKV(t *testing.T, tt serviceFromKVCase, service *bridge.Service, ok bool) {
	t.Helper()
	if ok != tt.want {
		t.Fatalf("ok = %t, want %t", ok, tt.want)
	}
	if !ok {
		if service != nil {
			t.Fatalf("expected nil service when ok is false, got: %+v", service)
		}
		return
	}
	if service.Name != tt.wantName {
		t.Fatalf("service.Name = %q, want %q", service.Name, tt.wantName)
	}
	if service.ID != tt.wantID {
		t.Fatalf("service.ID = %q, want %q", service.ID, tt.wantID)
	}
	if service.IP != tt.wantIP {
		t.Fatalf("service.IP = %q, want %q", service.IP, tt.wantIP)
	}
	if service.Port != tt.wantPort {
		t.Fatalf("service.Port = %d, want %d", service.Port, tt.wantPort)
	}
}
