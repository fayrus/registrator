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
	svc, err := b.newService(port, false)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Equal(t, "172.17.0.5", svc.IP)
}

func TestExecuteTagTemplate_Static(t *testing.T) {
	result, err := executeTagTemplate("web,api", minimalContainer("", ""))
	assert.NoError(t, err)
	assert.Equal(t, "web,api", result)
}

func TestExecuteTagTemplate_WithContainerField(t *testing.T) {
	c := minimalContainer("", "")
	c.Config.Hostname = "myhost"
	result, err := executeTagTemplate("host-{{.Config.Hostname}}", c)
	assert.NoError(t, err)
	assert.Equal(t, "host-myhost", result)
}

func TestExecuteTagTemplate_InvalidTemplate(t *testing.T) {
	_, err := executeTagTemplate("host-{{.Config.Hostname", minimalContainer("", ""))
	assert.Error(t, err)
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
	svc, err := b.newService(port, false)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Equal(t, "192.168.1.10", svc.IP)
}

func TestNewService_InvalidPortTagsOnlyFailAffectedService(t *testing.T) {
	b := newTestBridge(Config{})
	container := minimalContainer("192.168.1.10", "172.17.0.5")
	container.Config.Env = []string{
		"SERVICE_NAME=test-svc",
		"SERVICE_8080_TAGS=broken-{{.Config.Hostname",
		"SERVICE_9090_TAGS=ok",
	}

	invalidPort := ServicePort{
		HostIP:      "192.168.1.10",
		HostPort:    "32000",
		ExposedIP:   "172.17.0.5",
		ExposedPort: "8080",
		PortType:    "tcp",
		ContainerID: "abcdef1234567890",
		container:   container,
	}
	validPort := ServicePort{
		HostIP:      "192.168.1.10",
		HostPort:    "32001",
		ExposedIP:   "172.17.0.5",
		ExposedPort: "9090",
		PortType:    "tcp",
		ContainerID: "abcdef1234567890",
		container:   container,
	}

	invalidService, invalidErr := b.newService(invalidPort, true)
	validService, validErr := b.newService(validPort, true)

	assert.Nil(t, invalidService)
	assert.Error(t, invalidErr)
	assert.NoError(t, validErr)
	assert.NotNil(t, validService)
	assert.Equal(t, []string{"ok"}, validService.Tags)
}
