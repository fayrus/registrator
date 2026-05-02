package etcd2

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func init() {
	bridge.Register(new(Factory), "etcd2")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
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
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	}

	certFile := os.Getenv("ETCD_CERT_FILE")
	keyFile := os.Getenv("ETCD_KEY_FILE")
	caFile := os.Getenv("ETCD_CA_CERT_FILE")

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatal("etcd2: failed to load TLS keypair:", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		if caFile != "" {
			ca, err := os.ReadFile(caFile)
			if err != nil {
				log.Fatal("etcd2: failed to read CA cert:", err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(ca)
			tlsCfg.RootCAs = pool
		}
		cfg.TLS = tlsCfg
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		log.Fatal("etcd2: failed to connect:", err)
	}

	return &Etcd2Adapter{client: client, path: uri.Path}
}

type Etcd2Adapter struct {
	client *clientv3.Client
	path   string
}

func (r *Etcd2Adapter) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Status(ctx, r.client.Endpoints()[0])
	return err
}

func (r *Etcd2Adapter) Register(service *bridge.Service) error {
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
		log.Println("etcd2: failed to register service:", err)
	}
	return err
}

func (r *Etcd2Adapter) Deregister(service *bridge.Service) error {
	key := r.path + "/" + service.Name + "/" + service.ID

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.client.Delete(ctx, key)
	if err != nil {
		log.Println("etcd2: failed to deregister service:", err)
	}
	return err
}

func (r *Etcd2Adapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *Etcd2Adapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
