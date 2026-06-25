package consul

import (
	"errors"
	"net/url"
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	consulapi "github.com/hashicorp/consul/api"
)

type fakeKV struct {
	putErr   error
	delErr   error
	listErr  error
	pairs    consulapi.KVPairs
	lastKey  string
	lastVal  string
	lastList string
}

func (f *fakeKV) Put(p *consulapi.KVPair, q *consulapi.WriteOptions) (*consulapi.WriteMeta, error) {
	f.lastKey = p.Key
	f.lastVal = string(p.Value)
	return nil, f.putErr
}

func (f *fakeKV) Delete(key string, q *consulapi.WriteOptions) (*consulapi.WriteMeta, error) {
	f.lastKey = key
	return nil, f.delErr
}

func (f *fakeKV) List(prefix string, q *consulapi.QueryOptions) (consulapi.KVPairs, *consulapi.QueryMeta, error) {
	f.lastList = prefix
	return f.pairs, nil, f.listErr
}

func testSvc() *bridge.Service {
	return &bridge.Service{
		Name: "web",
		ID:   "web-1",
		IP:   "10.0.0.1",
		Port: 8080,
	}
}

func adapter(kv kvStore) *ConsulKVAdapter {
	return &ConsulKVAdapter{kv: kv, path: "/services"}
}

func TestRegister_BuildsCorrectPath(t *testing.T) {
	kv := &fakeKV{}
	if err := adapter(kv).Register(testSvc()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastKey != "services/web/web-1" {
		t.Errorf("unexpected key: %s", kv.lastKey)
	}
}

func TestRegister_StoresAddressAsValue(t *testing.T) {
	kv := &fakeKV{}
	if err := adapter(kv).Register(testSvc()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastVal != "10.0.0.1:8080" {
		t.Errorf("unexpected value: %s", kv.lastVal)
	}
}

func TestRegister_ReturnsErrorOnPutFail(t *testing.T) {
	kv := &fakeKV{putErr: errors.New("consul: connection refused")}
	if err := adapter(kv).Register(testSvc()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeregister_BuildsCorrectPath(t *testing.T) {
	kv := &fakeKV{}
	if err := adapter(kv).Deregister(testSvc()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastKey != "services/web/web-1" {
		t.Errorf("unexpected key: %s", kv.lastKey)
	}
}

func TestDeregister_ReturnsErrorOnDeleteFail(t *testing.T) {
	kv := &fakeKV{delErr: errors.New("consul: connection refused")}
	if err := adapter(kv).Deregister(testSvc()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_EmptyPath(t *testing.T) {
	kv := &fakeKV{}
	a := &ConsulKVAdapter{kv: kv, path: ""}
	if err := a.Register(testSvc()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastKey != "/web/web-1" {
		t.Errorf("unexpected key: %s", kv.lastKey)
	}
}

func TestRegister_RootPath(t *testing.T) {
	kv := &fakeKV{}
	a := &ConsulKVAdapter{kv: kv, path: "/"}
	if err := a.Register(testSvc()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastKey != "/web/web-1" {
		t.Errorf("unexpected key: %s", kv.lastKey)
	}
}

func TestNew_MalformedUnixURI(t *testing.T) {
	uri, _ := url.Parse("consulkv-unix:///var/run/consul.sock")
	_, err := new(Factory).New(uri)
	if err == nil {
		t.Fatal("expected error for malformed consulkv-unix URI, got nil")
	}
}

func TestServices_ListsRegisteredServices(t *testing.T) {
	kv := &fakeKV{pairs: consulapi.KVPairs{
		{Key: "services/web/host:web:80", Value: []byte("10.0.0.1:8080")},
		{Key: "services/api/host:api:9000", Value: []byte("[2001:db8::1]:9000")},
	}}
	services, err := adapter(kv).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kv.lastList != "services/" {
		t.Errorf("unexpected list prefix: %s", kv.lastList)
	}
	if len(services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(services))
	}
	if services[0].Name != "web" || services[0].ID != "host:web:80" || services[0].IP != "10.0.0.1" || services[0].Port != 8080 {
		t.Errorf("unexpected first service: %+v", services[0])
	}
	if services[1].Name != "api" || services[1].ID != "host:api:9000" || services[1].IP != "2001:db8::1" || services[1].Port != 9000 {
		t.Errorf("unexpected second service: %+v", services[1])
	}
}

func TestServices_SkipsMalformedEntries(t *testing.T) {
	kv := &fakeKV{pairs: consulapi.KVPairs{
		{Key: "services/web/host:web:80", Value: []byte("10.0.0.1:8080")},
		{Key: "services/missing-id", Value: []byte("10.0.0.2:8080")},
		{Key: "services/api/host:api:9000", Value: []byte("not-an-address")},
		{Key: "services/db/host:db:5432", Value: []byte("10.0.0.3:not-a-port")},
	}}
	services, err := adapter(kv).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(services))
	}
	if services[0].Name != "web" {
		t.Errorf("unexpected service: %+v", services[0])
	}
}

func TestServices_ReturnsErrorOnListFail(t *testing.T) {
	kv := &fakeKV{listErr: errors.New("consul: connection refused")}
	if _, err := adapter(kv).Services(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
