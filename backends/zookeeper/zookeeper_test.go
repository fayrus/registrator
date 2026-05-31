package zookeeper

import (
	"errors"
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	"github.com/samuel/go-zookeeper/zk"
)

type fakeZkClient struct {
	exists   map[string]bool
	existErr map[string]error
	createErr map[string]error
	deleteErr map[string]error
	children  map[string][]string
	childErr  map[string]error
}

func newFakeClient() *fakeZkClient {
	return &fakeZkClient{
		exists:    make(map[string]bool),
		existErr:  make(map[string]error),
		createErr: make(map[string]error),
		deleteErr: make(map[string]error),
		children:  make(map[string][]string),
		childErr:  make(map[string]error),
	}
}

func (f *fakeZkClient) Exists(path string) (bool, *zk.Stat, error) {
	return f.exists[path], nil, f.existErr[path]
}

func (f *fakeZkClient) Create(path string, data []byte, flags int32, acl []zk.ACL) (string, error) {
	if err := f.createErr[path]; err != nil {
		return "", err
	}
	f.exists[path] = true
	return path, nil
}

func (f *fakeZkClient) Delete(path string, version int32) error {
	if err := f.deleteErr[path]; err != nil {
		return err
	}
	delete(f.exists, path)
	return nil
}

func (f *fakeZkClient) Children(path string) ([]string, *zk.Stat, error) {
	return f.children[path], nil, f.childErr[path]
}

func testService() *bridge.Service {
	return &bridge.Service{
		Name: "web",
		IP:   "10.0.0.1",
		Port: 8080,
		Origin: bridge.ServicePort{
			ExposedPort: "8080",
			ContainerID: "abc123",
		},
	}
}

func adapter(client zkClient) *ZkAdapter {
	return &ZkAdapter{client: client, path: "/services"}
}

func TestRegister_CreatesBasePathAndServiceNode(t *testing.T) {
	c := newFakeClient()
	svc := testService()

	if err := adapter(c).Register(svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.exists["/services/web"] {
		t.Error("base path not created")
	}
	if !c.exists["/services/web/10.0.0.1:8080"] {
		t.Error("service node not created")
	}
}

func TestRegister_SkipsBasePathCreationIfExists(t *testing.T) {
	c := newFakeClient()
	c.exists["/services/web"] = true
	c.createErr["/services/web"] = errors.New("should not be called")

	if err := adapter(c).Register(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegister_ReturnsErrorOnExistsFail(t *testing.T) {
	c := newFakeClient()
	c.existErr["/services/web"] = errors.New("zk: connection closed")

	if err := adapter(c).Register(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_ReturnsErrorOnBasePathCreateFail(t *testing.T) {
	c := newFakeClient()
	c.createErr["/services/web"] = errors.New("zk: node exists")

	if err := adapter(c).Register(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_ReturnsErrorOnServiceNodeCreateFail(t *testing.T) {
	c := newFakeClient()
	c.createErr["/services/web/10.0.0.1:8080"] = errors.New("zk: node exists")

	if err := adapter(c).Register(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeregister_DeletesServiceNodeAndBasePath(t *testing.T) {
	c := newFakeClient()
	c.exists["/services/web"] = true
	c.exists["/services/web/10.0.0.1:8080"] = true
	c.children["/services/web"] = []string{}

	if err := adapter(c).Deregister(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.exists["/services/web"] {
		t.Error("base path should have been deleted")
	}
}

func TestDeregister_KeepsBasePathIfOtherChildrenExist(t *testing.T) {
	c := newFakeClient()
	c.exists["/services/web"] = true
	c.exists["/services/web/10.0.0.1:8080"] = true
	c.children["/services/web"] = []string{"10.0.0.2:8080"}

	if err := adapter(c).Deregister(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.exists["/services/web"] {
		t.Error("base path should have been kept")
	}
}

func TestDeregister_ReturnsErrorIfDeleteFails(t *testing.T) {
	c := newFakeClient()
	c.deleteErr["/services/web/10.0.0.1:8080"] = errors.New("zk: no node")

	if err := adapter(c).Deregister(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPing_Success(t *testing.T) {
	c := newFakeClient()
	c.exists["/"] = true

	if err := adapter(c).Ping(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPing_ReturnsErrorOnFail(t *testing.T) {
	c := newFakeClient()
	c.existErr["/"] = errors.New("zk: connection closed")

	if err := adapter(c).Ping(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
