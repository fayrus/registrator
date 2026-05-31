package consul

import (
	"errors"
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	consulapi "github.com/hashicorp/consul/api"
)

type fakeKV struct {
	putErr  error
	delErr  error
	lastKey string
	lastVal string
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
