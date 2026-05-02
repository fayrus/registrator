package coredns

import (
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	"github.com/stretchr/testify/assert"
)

func newTestAdapter(zone string) *CoreDNSAdapter {
	return &CoreDNSAdapter{prefix: "/skydns", zone: zone}
}

func newTestService(id, name, ip string, port int) *bridge.Service {
	return &bridge.Service{ID: id, Name: name, IP: ip, Port: port}
}

func TestReverseDomain(t *testing.T) {
	assert.Equal(t, "local/service", reverseDomain("service.local"))
	assert.Equal(t, "com/example/api", reverseDomain("api.example.com"))
	assert.Equal(t, "local", reverseDomain("local"))
}

func TestServiceKey(t *testing.T) {
	a := newTestAdapter("service.local")
	svc := newTestService("host1:whoami:8080", "whoami", "1.2.3.4", 8080)
	key := a.serviceKey(svc)
	assert.Equal(t, "/skydns/local/service/whoami/host1-whoami-8080", key)
}

func TestServiceKey_CustomPrefix(t *testing.T) {
	a := &CoreDNSAdapter{prefix: "/dns", zone: "service.local"}
	svc := newTestService("host:svc:80", "api", "10.0.0.1", 80)
	key := a.serviceKey(svc)
	assert.Equal(t, "/dns/local/service/api/host-svc-80", key)
}
