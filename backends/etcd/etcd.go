package etcd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	"github.com/fayrus/registrator/internal/etcdtls"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	bridge.Register(new(Factory), "etcd")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	client, err := etcdtls.NewClient(uri.Host)
	if err != nil {
		return nil, fmt.Errorf("etcd: %w", err)
	}
	return &EtcdAdapter{client: client, path: uri.Path}, nil
}

type etcdClient interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error)
	Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error)
	Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error)
	Endpoints() []string
}

type EtcdAdapter struct {
	client etcdClient
	path   string
}

func (r *EtcdAdapter) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Status(ctx, r.client.Endpoints()[0])
	return err
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	key := r.path + "/" + service.Name + "/" + service.ID
	value := net.JoinHostPort(service.IP, strconv.Itoa(service.Port))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	if service.TTL > 0 {
		lease, lerr := r.client.Grant(ctx, int64(service.TTL))
		if lerr != nil {
			return lerr
		}
		_, err = r.client.Put(ctx, key, value, clientv3.WithLease(lease.ID))
	} else {
		_, err = r.client.Put(ctx, key, value)
	}

	if err != nil {
		log.Println("etcd: failed to register service:", err)
	}
	return err
}

func (r *EtcdAdapter) Deregister(service *bridge.Service) error {
	key := r.path + "/" + service.Name + "/" + service.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.client.Delete(ctx, key)
	if err != nil {
		log.Println("etcd: failed to deregister service:", err)
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	prefix := r.servicePrefix()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return []*bridge.Service{}, err
	}
	services := make([]*bridge.Service, 0, len(res.Kvs))
	for _, kv := range res.Kvs {
		service, ok := serviceFromKV(prefix, string(kv.Key), string(kv.Value))
		if ok {
			services = append(services, service)
		}
	}
	return services, nil
}

func (r *EtcdAdapter) servicePrefix() string {
	return r.path + "/"
}

func serviceFromKV(prefix, key, value string) (*bridge.Service, bool) {
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
