package consul

import (
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
)

func newTestService(attrs map[string]string) *bridge.Service {
	return &bridge.Service{
		ID:   "test-id",
		Name: "test-svc",
		IP:   "127.0.0.1",
		Port: 8080,
		Attrs: attrs,
		Origin: bridge.ServicePort{
			ContainerID: "abcdef123456789",
			ExposedPort: "8080/tcp",
		},
	}
}

func adapter() *ConsulAdapter {
	return &ConsulAdapter{client: &consulapi.Client{}}
}

func TestBuildCheck_HTTP(t *testing.T) {
	svc := newTestService(map[string]string{
		"check_http":     "/health",
		"check_interval": "15s",
		"check_timeout":  "5s",
	})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "http://127.0.0.1:8080/health", check.HTTP)
	assert.Equal(t, "15s", check.Interval)
	assert.Equal(t, "5s", check.Timeout)
}

func TestBuildCheck_HTTPS(t *testing.T) {
	svc := newTestService(map[string]string{
		"check_https":           "/health",
		"check_tls_skip_verify": "true",
	})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "https://127.0.0.1:8080/health", check.HTTP)
	assert.True(t, check.TLSSkipVerify)
	assert.Equal(t, DefaultInterval, check.Interval)
}

func TestBuildCheck_TCP(t *testing.T) {
	svc := newTestService(map[string]string{"check_tcp": "true"})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "127.0.0.1:8080", check.TCP)
}

func TestBuildCheck_TTL(t *testing.T) {
	svc := newTestService(map[string]string{"check_ttl": "30s"})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "30s", check.TTL)
}

func TestBuildCheck_Script(t *testing.T) {
	svc := newTestService(map[string]string{
		"check_script":   "/bin/check.sh arg1 arg2",
		"check_interval": "10s",
	})
	check := adapter().buildCheck(svc)
	assert.Equal(t, []string{"/bin/check.sh", "arg1", "arg2"}, check.Args)
}

func TestBuildCheck_GRPC(t *testing.T) {
	svc := newTestService(map[string]string{
		"check_grpc":            "true",
		"check_grpc_use_tls":    "true",
		"check_tls_skip_verify": "true",
	})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "127.0.0.1:8080", check.GRPC)
	assert.True(t, check.GRPCUseTLS)
	assert.True(t, check.TLSSkipVerify)
}

func TestBuildCheck_None(t *testing.T) {
	svc := newTestService(map[string]string{})
	check := adapter().buildCheck(svc)
	assert.Nil(t, check)
}

func TestBuildCheck_InitialStatus(t *testing.T) {
	svc := newTestService(map[string]string{
		"check_http":           "/health",
		"check_initial_status": "passing",
	})
	check := adapter().buildCheck(svc)
	assert.Equal(t, "passing", check.Status)
}

func TestEnableTagOverride_True(t *testing.T) {
	svc := newTestService(map[string]string{"enable_tag_override": "true"})
	assert.Equal(t, "true", svc.Attrs["enable_tag_override"])
	enabled := svc.Attrs["enable_tag_override"] == "true"
	assert.True(t, enabled)
}

func TestEnableTagOverride_False(t *testing.T) {
	svc := newTestService(map[string]string{})
	enabled := svc.Attrs["enable_tag_override"] == "true"
	assert.False(t, enabled)
}
