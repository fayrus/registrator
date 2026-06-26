package etcdlegacy

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	etcd2 "github.com/coreos/go-etcd/etcd"
	"github.com/fayrus/registrator/internal/bridge"
	"github.com/fayrus/registrator/internal/kvutil"
	etcd "gopkg.in/coreos/go-etcd.v0/etcd"
)

func init() {
	bridge.Register(new(Factory), "etcd-legacy")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	urls := make([]string, 0)
	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	} else {
		urls = append(urls, "http://127.0.0.1:2379")
	}

	res, err := http.Get(urls[0] + "/version")
	if err != nil {
		return nil, fmt.Errorf("etcd-legacy: error retrieving version: %w", err)
	}

	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)

	if match, _ := regexp.Match("0\\.4\\.*", body); match {
		log.Println("etcd-legacy: using v0 client")
		return &EtcdAdapter{client: etcd.NewClient(urls), path: uri.Path}, nil
	}

	return &EtcdAdapter{client2: etcd2.NewClient(urls), path: uri.Path}, nil
}

type EtcdAdapter struct {
	client  etcdV0Client
	client2 etcdV2Client

	path string
}

type etcdV0Client interface {
	Set(key string, value string, ttl uint64) (*etcd.Response, error)
	Delete(key string, recursive bool) (*etcd.Response, error)
	Get(key string, sort, recursive bool) (*etcd.Response, error)
	SendRequest(rr *etcd.RawRequest) (*etcd.RawResponse, error)
	SyncCluster() bool
}

type etcdV2Client interface {
	Set(key string, value string, ttl uint64) (*etcd2.Response, error)
	Delete(key string, recursive bool) (*etcd2.Response, error)
	Get(key string, sort, recursive bool) (*etcd2.Response, error)
	SendRequest(rr *etcd2.RawRequest) (*etcd2.RawResponse, error)
	SyncCluster() bool
}

func (r *EtcdAdapter) Ping() error {
	r.syncEtcdCluster()

	var err error
	if r.client != nil {
		rr := etcd.NewRawRequest("GET", "version", nil, nil)
		_, err = r.client.SendRequest(rr)
	} else {
		rr := etcd2.NewRawRequest("GET", "version", nil, nil)
		_, err = r.client2.SendRequest(rr)
	}

	if err != nil {
		return err
	}
	return nil
}

func (r *EtcdAdapter) syncEtcdCluster() {
	var result bool
	if r.client != nil {
		result = r.client.SyncCluster()
	} else {
		result = r.client2.SyncCluster()
	}

	if !result {
		log.Println("etcd-legacy: sync cluster was unsuccessful")
	}
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)

	var err error
	if r.client != nil {
		_, err = r.client.Set(path, addr, uint64(service.TTL))
	} else {
		_, err = r.client2.Set(path, addr, uint64(service.TTL))
	}

	if err != nil {
		log.Println("etcd-legacy: failed to register service:", err)
	}
	return err
}

func (r *EtcdAdapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID

	var err error
	if r.client != nil {
		_, err = r.client.Delete(path, false)
	} else {
		_, err = r.client2.Delete(path, false)
	}

	if err != nil {
		log.Println("etcd-legacy: failed to deregister service:", err)
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	r.syncEtcdCluster()

	if r.client != nil {
		res, err := r.client.Get(r.path, true, true)
		if err != nil {
			return []*bridge.Service{}, err
		}
		return servicesFromV0Node(r.servicePrefix(), res.Node), nil
	}

	res, err := r.client2.Get(r.path, true, true)
	if err != nil {
		return []*bridge.Service{}, err
	}
	return servicesFromV2Node(r.servicePrefix(), res.Node), nil
}

func (r *EtcdAdapter) servicePrefix() string {
	return r.path + "/"
}

func servicesFromV0Node(prefix string, node *etcd.Node) []*bridge.Service {
	if node == nil {
		return []*bridge.Service{}
	}
	services := make([]*bridge.Service, 0, len(node.Nodes))
	appendV0Services(prefix, node, &services)
	return services
}

func appendV0Services(prefix string, node *etcd.Node, services *[]*bridge.Service) {
	if node.Dir {
		for _, child := range node.Nodes {
			appendV0Services(prefix, child, services)
		}
		return
	}
	service, ok := kvutil.ServiceFromKV(prefix, node.Key, node.Value)
	if ok {
		*services = append(*services, service)
	}
}

func servicesFromV2Node(prefix string, node *etcd2.Node) []*bridge.Service {
	if node == nil {
		return []*bridge.Service{}
	}
	services := make([]*bridge.Service, 0, len(node.Nodes))
	appendV2Services(prefix, node, &services)
	return services
}

func appendV2Services(prefix string, node *etcd2.Node, services *[]*bridge.Service) {
	if node.Dir {
		for _, child := range node.Nodes {
			appendV2Services(prefix, child, services)
		}
		return
	}
	service, ok := kvutil.ServiceFromKV(prefix, node.Key, node.Value)
	if ok {
		*services = append(*services, service)
	}
}
