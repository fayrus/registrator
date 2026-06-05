package etcdtls

import (
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// NewClient builds an etcd v3 client from the given host and environment.
// Additional endpoints are read from ETCD_ENDPOINTS (comma-separated).
// TLS is configured from ETCD_CERT_FILE, ETCD_KEY_FILE, ETCD_CA_CERT_FILE.
func NewClient(host string) (*clientv3.Client, error) {
	cfg := clientv3.Config{
		Endpoints:   buildEndpoints(host),
		DialTimeout: 5 * time.Second,
	}

	tlsCfg, err := Build(
		os.Getenv("ETCD_CERT_FILE"),
		os.Getenv("ETCD_KEY_FILE"),
		os.Getenv("ETCD_CA_CERT_FILE"),
	)
	if err != nil {
		return nil, err
	}
	if tlsCfg != nil {
		cfg.TLS = tlsCfg
	}

	return clientv3.New(cfg)
}

func buildEndpoints(host string) []string {
	var endpoints []string
	if host != "" {
		endpoints = append(endpoints, host)
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
	return endpoints
}
