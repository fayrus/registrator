package bridge

import (
	"sort"
	"testing"

	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
)

func configWithEnv(env ...string) *dockerapi.Config {
	return &dockerapi.Config{Env: env}
}

func TestEscapedComma(t *testing.T) {
	cases := []struct {
		Tag      string
		Expected []string
	}{
		{
			Tag:      "",
			Expected: []string{},
		},
		{
			Tag:      "foobar",
			Expected: []string{"foobar"},
		},
		{
			Tag:      "foo,bar",
			Expected: []string{"foo", "bar"},
		},
		{
			Tag:      "foo\\,bar",
			Expected: []string{"foo,bar"},
		},
		{
			Tag:      "foo,bar\\,baz",
			Expected: []string{"foo", "bar,baz"},
		},
		{
			Tag:      "\\,foobar\\,",
			Expected: []string{",foobar,"},
		},
		{
			Tag:      ",,,,foo,,,bar,,,",
			Expected: []string{"foo", "bar"},
		},
		{
			Tag:      ",,,,",
			Expected: []string{},
		},
		{
			Tag:      ",,\\,,",
			Expected: []string{","},
		},
	}

	for _, c := range cases {
		results := recParseEscapedComma(c.Tag)
		sort.Strings(c.Expected)
		sort.Strings(results)
		assert.EqualValues(t, c.Expected, results)
	}
}

func TestServiceMetaData_Generic(t *testing.T) {
	cfg := configWithEnv("SERVICE_NAME=web", "SERVICE_TAGS=a,b")
	meta, _ := serviceMetaData(cfg, "8080", "tcp")
	assert.Equal(t, "web", meta["name"])
	assert.Equal(t, "a,b", meta["tags"])
}

func TestServiceMetaData_PortSpecific(t *testing.T) {
	cfg := configWithEnv("SERVICE_8080_NAME=api", "SERVICE_9000_NAME=other")
	meta, fromPort := serviceMetaData(cfg, "8080", "tcp")
	assert.Equal(t, "api", meta["name"])
	assert.True(t, fromPort["name"])
	_, hasOther := meta["9000_name"]
	assert.False(t, hasOther)
}

func TestServiceMetaData_ProtocolSpecific_Match(t *testing.T) {
	cfg := configWithEnv("SERVICE_8080_tcp_NAME=tcp-svc")
	meta, fromPort := serviceMetaData(cfg, "8080", "tcp")
	assert.Equal(t, "tcp-svc", meta["name"])
	assert.True(t, fromPort["name"])
}

func TestServiceMetaData_ProtocolSpecific_NoMatch(t *testing.T) {
	cfg := configWithEnv("SERVICE_8080_tcp_NAME=tcp-svc")
	meta, _ := serviceMetaData(cfg, "8080", "udp")
	_, exists := meta["name"]
	assert.False(t, exists)
}

func TestServiceMetaData_ProtocolSpecific_OverridesGeneric(t *testing.T) {
	// Protocol-specific takes precedence over port-generic for same port
	cfg := configWithEnv("SERVICE_8080_NAME=generic", "SERVICE_8080_tcp_NAME=specific")
	meta, _ := serviceMetaData(cfg, "8080", "tcp")
	assert.Equal(t, "specific", meta["name"])
}
