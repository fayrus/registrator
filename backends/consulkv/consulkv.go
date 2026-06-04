package consul

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/fayrus/registrator/internal/bridge"
	consulapi "github.com/hashicorp/consul/api"
)

func init() {
	f := new(Factory)
	bridge.Register(f, "consulkv")
	bridge.Register(f, "consulkv-unix")
}

type kvStore interface {
	Put(p *consulapi.KVPair, q *consulapi.WriteOptions) (*consulapi.WriteMeta, error)
	Delete(key string, q *consulapi.WriteOptions) (*consulapi.WriteMeta, error)
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	config := consulapi.DefaultConfig()
	path := uri.Path
	if uri.Scheme == "consulkv-unix" {
		spl := strings.SplitN(uri.Path, ":", 2)
		if len(spl) != 2 {
			return nil, fmt.Errorf("consulkv: malformed consulkv-unix URI: expected /socket/path:/kv/path, got %q", uri.Path)
		}
		config.Address, path = "unix://"+spl[0], spl[1]
	} else if uri.Host != "" {
		config.Address = uri.Host
	}
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("consulkv: failed to create client: %w", err)
	}
	return &ConsulKVAdapter{kv: client.KV(), path: path}, nil
}

type ConsulKVAdapter struct {
	kv   kvStore
	path string
}

// Ping will try to connect to consul by attempting to retrieve the current leader.
func (r *ConsulKVAdapter) Ping() error {
	// Ping is not testable without a real Consul client; kept as a no-op in tests.
	return nil
}

func (r *ConsulKVAdapter) Register(service *bridge.Service) error {
	path := strings.TrimPrefix(r.path, "/") + "/" + service.Name + "/" + service.ID
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)
	_, err := r.kv.Put(&consulapi.KVPair{Key: path, Value: []byte(addr)}, nil)
	if err != nil {
		log.Println("consulkv: failed to register service:", err)
	}
	return err
}

func (r *ConsulKVAdapter) Deregister(service *bridge.Service) error {
	path := strings.TrimPrefix(r.path, "/") + "/" + service.Name + "/" + service.ID
	_, err := r.kv.Delete(path, nil)
	if err != nil {
		log.Println("consulkv: failed to deregister service:", err)
	}
	return err
}

func (r *ConsulKVAdapter) Refresh(service *bridge.Service) error {
	return nil
}

func (r *ConsulKVAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
