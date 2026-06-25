package zookeeper

import (
	"encoding/json"
	"log"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	"github.com/samuel/go-zookeeper/zk"
)

func init() {
	bridge.Register(new(Factory), "zookeeper")
}

type zkClient interface {
	Exists(path string) (bool, *zk.Stat, error)
	Create(path string, data []byte, flags int32, acl []zk.ACL) (string, error)
	Delete(path string, version int32) error
	Children(path string) ([]string, *zk.Stat, error)
	Get(path string) ([]byte, *zk.Stat, error)
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	c, _, err := zk.Connect([]string{uri.Host}, (time.Second * 10))
	if err != nil {
		return nil, err
	}
	exists, _, err := c.Exists(uri.Path)
	if err != nil {
		return nil, err
	}
	if !exists {
		_, err = c.Create(uri.Path, []byte{}, 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return nil, err
		}
	}
	return &ZkAdapter{client: c, path: uri.Path}, nil
}

type ZkAdapter struct {
	client zkClient
	path   string
}

type ZnodeBody struct {
	ID          string
	Name        string
	IP          string
	PublicPort  int
	PrivatePort int
	ContainerID string
	Tags        []string
	Attrs       map[string]string
}

func (r *ZkAdapter) Register(service *bridge.Service) error {
	privatePort, _ := strconv.Atoi(service.Origin.ExposedPort)
	publicPortString := strconv.Itoa(service.Port)
	acl := zk.WorldACL(zk.PermAll)
	basePath := r.servicePath(service.Name)

	exists, _, err := r.client.Exists(basePath)
	if err != nil {
		return err
	}
	if !exists {
		_, err = r.client.Create(basePath, []byte{}, 0, acl)
		if err != nil {
			return err
		}
	}

	zbody := &ZnodeBody{ID: service.ID, Name: service.Name, IP: service.IP, PublicPort: service.Port, PrivatePort: privatePort, Tags: service.Tags, Attrs: service.Attrs, ContainerID: service.Origin.ContainerID}
	body, err := json.Marshal(zbody)
	if err != nil {
		return err
	}

	path := basePath + "/" + service.IP + ":" + publicPortString
	_, err = r.client.Create(path, body, 0, acl)
	return err
}

func (r *ZkAdapter) Ping() error {
	_, _, err := r.client.Exists("/")
	if err != nil {
		log.Println("zookeeper: error on ping check for Exists(/): ", err)
		return err
	}
	return nil
}

func (r *ZkAdapter) Deregister(service *bridge.Service) error {
	basePath := r.servicePath(service.Name)
	publicPortString := strconv.Itoa(service.Port)
	servicePortPath := basePath + "/" + service.IP + ":" + publicPortString

	if err := r.client.Delete(servicePortPath, -1); err != nil {
		return err
	}

	children, _, err := r.client.Children(basePath)
	if err != nil {
		return err
	}
	if len(children) == 0 {
		err = r.client.Delete(basePath, -1)
		if err != nil {
			log.Println("zookeeper: failed to delete service path:", err)
		}
	}
	return err
}

func (r *ZkAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *ZkAdapter) Services() ([]*bridge.Service, error) {
	serviceNames, _, err := r.client.Children(r.path)
	if err != nil {
		return []*bridge.Service{}, err
	}
	services := make([]*bridge.Service, 0)
	for _, serviceName := range serviceNames {
		basePath := r.servicePath(serviceName)
		serviceNodes, _, err := r.client.Children(basePath)
		if err != nil {
			return []*bridge.Service{}, err
		}
		for _, serviceNode := range serviceNodes {
			body, _, err := r.client.Get(path.Join(basePath, serviceNode))
			if err != nil {
				return []*bridge.Service{}, err
			}
			service, ok := serviceFromZnode(body)
			if ok {
				services = append(services, service)
			}
		}
	}
	return services, nil
}

func (r *ZkAdapter) servicePath(serviceName string) string {
	if r.path == "/" {
		return r.path + serviceName
	}
	return r.path + "/" + serviceName
}

func serviceFromZnode(body []byte) (*bridge.Service, bool) {
	var zbody ZnodeBody
	if err := json.Unmarshal(body, &zbody); err != nil {
		return nil, false
	}
	if zbody.ID == "" || zbody.Name == "" || zbody.IP == "" || zbody.PublicPort <= 0 {
		return nil, false
	}
	return &bridge.Service{
		ID:    zbody.ID,
		Name:  zbody.Name,
		IP:    zbody.IP,
		Port:  zbody.PublicPort,
		Tags:  zbody.Tags,
		Attrs: zbody.Attrs,
	}, true
}
