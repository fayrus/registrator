package coredns

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func init() {
	bridge.Register(new(Factory), "coredns")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	endpoints := []string{}
	if uri.Host != "" {
		endpoints = append(endpoints, uri.Host)
	}
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

	// DNS zone from query param, default to "local"
	zone := uri.Query().Get("zone")
	if zone == "" {
		zone = "local"
	}

	// etcd key prefix from path, default to "/skydns"
	prefix := uri.Path
	if prefix == "" || prefix == "/" {
		prefix = "/skydns"
	}

	cfg := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	}

	certFile := os.Getenv("ETCD_CERT_FILE")
	keyFile := os.Getenv("ETCD_KEY_FILE")
	caFile := os.Getenv("ETCD_CA_CERT_FILE")
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatal("coredns: failed to load TLS keypair:", err)
		}
		tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		if caFile != "" {
			ca, err := os.ReadFile(caFile)
			if err != nil {
				log.Fatal("coredns: failed to read CA cert:", err)
			}
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(ca)
			tlsCfg.RootCAs = pool
		}
		cfg.TLS = tlsCfg
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		log.Fatal("coredns: failed to connect to etcd:", err)
	}

	log.Printf("coredns: using zone=%s prefix=%s endpoints=%v", zone, prefix, endpoints)
	return &CoreDNSAdapter{client: client, prefix: prefix, zone: zone}
}

type CoreDNSAdapter struct {
	client *clientv3.Client
	prefix string
	zone   string
}

// skydnsRecord is the JSON value stored in etcd for CoreDNS etcd plugin.
type skydnsRecord struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// serviceKey builds the etcd key in SkyDNS format:
// <prefix>/<reversed-zone>/<service-name>/<service-id>
func (r *CoreDNSAdapter) serviceKey(service *bridge.Service) string {
	reversedZone := reverseDomain(r.zone)
	// sanitize service ID — replace colons and dots with dashes for valid DNS labels
	safeID := strings.NewReplacer(":", "-", ".", "-").Replace(service.ID)
	return fmt.Sprintf("%s/%s/%s/%s", r.prefix, reversedZone, service.Name, safeID)
}

func reverseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, "/")
}

func (r *CoreDNSAdapter) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Status(ctx, r.client.Endpoints()[0])
	return err
}

func (r *CoreDNSAdapter) Register(service *bridge.Service) error {
	key := r.serviceKey(service)
	record := skydnsRecord{Host: service.IP, Port: service.Port}
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if service.TTL > 0 {
		lease, lerr := r.client.Grant(ctx, int64(service.TTL))
		if lerr != nil {
			return lerr
		}
		_, err = r.client.Put(ctx, key, string(value), clientv3.WithLease(lease.ID))
	} else {
		_, err = r.client.Put(ctx, key, string(value))
	}

	if err != nil {
		log.Println("coredns: failed to register service:", err)
	}
	return err
}

func (r *CoreDNSAdapter) Deregister(service *bridge.Service) error {
	key := r.serviceKey(service)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.client.Delete(ctx, key)
	if err != nil {
		log.Println("coredns: failed to deregister service:", err)
	}
	return err
}

func (r *CoreDNSAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *CoreDNSAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
