package kvutil

import "testing"

func TestServiceFromKV(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		want     bool
		wantName string
		wantID   string
		wantIP   string
		wantPort int
	}{
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
			if ok != tt.want {
				t.Fatalf("ok = %t, want %t", ok, tt.want)
			}
			if !ok {
				return
			}
			if service.Name != tt.wantName || service.ID != tt.wantID || service.IP != tt.wantIP || service.Port != tt.wantPort {
				t.Fatalf("unexpected service: %+v", service)
			}
		})
	}
}
