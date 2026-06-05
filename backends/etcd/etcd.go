package etcd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
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
	endpoints := []string{}

	if uri.Host != "" {
		endpoints = append(endpoints, uri.Host)
	}

	// Support additional endpoints via ETCD_ENDPOINTS env var (comma-separated)
	if env := os.Getenv("ETCD_ENDPOINTS"); env != "" {
		for _, ep := range strings.Split(env, ",") {
			if ep = strings.TrimSpace(ep); ep != "" {
				endpoints = append(endpoints, ep)
			}
		}
	}

	if len(endpoints) == 0 {
		endpoints = []string{"127.0.0.1:2379"}
	}

	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}

	tlsCfg, err := etcdtls.Build(
		os.Getenv("ETCD_CERT_FILE"),
		os.Getenv("ETCD_KEY_FILE"),
		os.Getenv("ETCD_CA_CERT_FILE"),
	)
	if err != nil {
		return nil, fmt.Errorf("etcd: %w", err)
	}
	if tlsCfg != nil {
		cfg.TLS = tlsCfg
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("etcd: failed to connect: %w", err)
	}

	return &EtcdAdapter{client: client, path: uri.Path}, nil
}

type etcdClient interface {
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
	return []*bridge.Service{}, nil
}
