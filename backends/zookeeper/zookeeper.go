package zookeeper

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	"github.com/samuel/go-zookeeper/zk"
)

func init() {
	bridge.Register(new(Factory), "zookeeper")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	c, _, err := zk.Connect([]string{uri.Host}, (time.Second * 10))
	if err != nil {
		return nil, err
	}
	exists, _, err := c.Exists(uri.Path)
	if err != nil {
		log.Println("zookeeper: error checking if base path exists:", err)
	}
	if !exists {
		c.Create(uri.Path, []byte{}, 0, zk.WorldACL(zk.PermAll))
	}
	return &ZkAdapter{client: c, path: uri.Path}, nil
}

type ZkAdapter struct {
	client *zk.Conn
	path   string
}

type ZnodeBody struct {
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
	basePath := r.path + "/" + service.Name
	if (r.path == "/") {
		basePath = r.path + service.Name
	}

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

	zbody := &ZnodeBody{Name: service.Name, IP: service.IP, PublicPort: service.Port, PrivatePort: privatePort, Tags: service.Tags, Attrs: service.Attrs, ContainerID: service.Origin.ContainerID}
	body, err := json.Marshal(zbody)
	if err != nil {
		return err
	}

	path := basePath + "/" + service.IP + ":" + publicPortString
	_, err = r.client.Create(path, body, 1, acl)
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
	basePath := r.path + "/" + service.Name
	if (r.path == "/") {
		basePath = r.path + service.Name
	}
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
	return []*bridge.Service{}, nil
}
