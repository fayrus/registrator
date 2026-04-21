package bridge

import (
	"testing"

	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	bridge, err := New(nil, "", Config{})
	assert.Nil(t, bridge)
	assert.Error(t, err)
}

func TestNewValid(t *testing.T) {
	Register(new(fakeFactory), "fake")
	// Note: the following is valid for New() since it does not
	// actually connect to docker.
	bridge, err := New(nil, "fake://", Config{})

	assert.NotNil(t, bridge)
	assert.NoError(t, err)
}

func newTestBridge(config Config) *Bridge {
	return &Bridge{
		config:         config,
		registry:       &fakeAdapter{},
		services:       make(map[string][]*Service),
		deadContainers: make(map[string]*DeadContainer),
	}
}

func minimalContainer(hostIP, containerIP string) *dockerapi.Container {
	return &dockerapi.Container{
		ID:   "abcdef1234567890",
		Name: "/test-container",
		Config: &dockerapi.Config{
			Image: "test-image",
			Env:   []string{"SERVICE_NAME=test-svc"},
		},
		HostConfig: &dockerapi.HostConfig{
			NetworkMode: "bridge",
		},
		NetworkSettings: &dockerapi.NetworkSettings{
			IPAddress: containerIP,
			Ports:     map[dockerapi.Port][]dockerapi.PortBinding{},
		},
	}
}

func TestIpFromContainer_UsesContainerIP(t *testing.T) {
	b := newTestBridge(Config{IpFromContainer: true})
	port := ServicePort{
		HostIP:      "192.168.1.10",
		HostPort:    "32000",
		ExposedIP:   "172.17.0.5",
		ExposedPort: "8080",
		PortType:    "tcp",
		ContainerID: "abcdef1234567890",
		container:   minimalContainer("192.168.1.10", "172.17.0.5"),
	}
	svc := b.newService(port, false)
	assert.NotNil(t, svc)
	assert.Equal(t, "172.17.0.5", svc.IP)
}

func TestIpFromContainer_Disabled_UsesHostIP(t *testing.T) {
	b := newTestBridge(Config{IpFromContainer: false})
	port := ServicePort{
		HostIP:      "192.168.1.10",
		HostPort:    "32000",
		ExposedIP:   "172.17.0.5",
		ExposedPort: "8080",
		PortType:    "tcp",
		ContainerID: "abcdef1234567890",
		container:   minimalContainer("192.168.1.10", "172.17.0.5"),
	}
	svc := b.newService(port, false)
	assert.NotNil(t, svc)
	assert.Equal(t, "192.168.1.10", svc.IP)
}
