package kvutil

import (
	"net"
	"strconv"
	"strings"

	"github.com/fayrus/registrator/internal/bridge"
)

func ServiceFromKV(prefix, key, value string) (*bridge.Service, bool) {
	if !strings.HasPrefix(key, prefix) {
		return nil, false
	}
	relativeKey := strings.TrimPrefix(key, prefix)
	keyParts := strings.SplitN(relativeKey, "/", 2)
	if len(keyParts) != 2 || keyParts[0] == "" || keyParts[1] == "" {
		return nil, false
	}
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return nil, false
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return nil, false
	}
	return &bridge.Service{
		Name: keyParts[0],
		ID:   keyParts[1],
		IP:   host,
		Port: port,
	}, true
}
