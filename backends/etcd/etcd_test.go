package etcd

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	mvccpb "go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type fakeEtcdClient struct {
	putErr    error
	delErr    error
	getErr    error
	grantErr  error
	pingErr   error
	kvs       []*mvccpb.KeyValue
	lastGet   string
	lastKey   string
	lastValue string
	lastTTL   int64
}

func (f *fakeEtcdClient) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	f.lastGet = key
	return &clientv3.GetResponse{Kvs: f.kvs}, f.getErr
}

func (f *fakeEtcdClient) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	f.lastKey = key
	f.lastValue = val
	return nil, f.putErr
}

func (f *fakeEtcdClient) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	f.lastKey = key
	return nil, f.delErr
}

func (f *fakeEtcdClient) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	f.lastTTL = ttl
	if f.grantErr != nil {
		return nil, f.grantErr
	}
	return &clientv3.LeaseGrantResponse{ID: 1}, nil
}

func (f *fakeEtcdClient) Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error) {
	return nil, f.pingErr
}

func (f *fakeEtcdClient) Endpoints() []string {
	return []string{"127.0.0.1:2379"}
}

func testService() *bridge.Service {
	return &bridge.Service{
		Name: "web",
		ID:   "web-1",
		IP:   "10.0.0.1",
		Port: 8080,
	}
}

func adapter(c etcdClient) *EtcdAdapter {
	return &EtcdAdapter{client: c, path: "/services"}
}

func TestRegister_BuildsCorrectKeyAndValue(t *testing.T) {
	c := &fakeEtcdClient{}
	if err := adapter(c).Register(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastKey != "/services/web/web-1" {
		t.Errorf("unexpected key: %s", c.lastKey)
	}
	if c.lastValue != "10.0.0.1:8080" {
		t.Errorf("unexpected value: %s", c.lastValue)
	}
}

func TestRegister_ReturnsErrorOnPutFail(t *testing.T) {
	c := &fakeEtcdClient{putErr: errors.New("etcd: connection refused")}
	if err := adapter(c).Register(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_UsesLeaseWhenTTLSet(t *testing.T) {
	c := &fakeEtcdClient{}
	svc := testService()
	svc.TTL = 30
	if err := adapter(c).Register(svc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastTTL != 30 {
		t.Errorf("expected TTL 30, got %d", c.lastTTL)
	}
}

func TestRegister_ReturnsErrorOnGrantFail(t *testing.T) {
	c := &fakeEtcdClient{grantErr: errors.New("etcd: lease error")}
	svc := testService()
	svc.TTL = 30
	if err := adapter(c).Register(svc); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeregister_BuildsCorrectKey(t *testing.T) {
	c := &fakeEtcdClient{}
	if err := adapter(c).Deregister(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastKey != "/services/web/web-1" {
		t.Errorf("unexpected key: %s", c.lastKey)
	}
}

func TestDeregister_ReturnsErrorOnDeleteFail(t *testing.T) {
	c := &fakeEtcdClient{delErr: errors.New("etcd: connection refused")}
	if err := adapter(c).Deregister(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRefresh_DelegatesToRegister(t *testing.T) {
	c := &fakeEtcdClient{}
	if err := adapter(c).Refresh(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastKey != "/services/web/web-1" {
		t.Errorf("unexpected key: %s", c.lastKey)
	}
}

func TestServices_ListsRegisteredServices(t *testing.T) {
	c := &fakeEtcdClient{kvs: []*mvccpb.KeyValue{
		{Key: []byte("/services/web/host:web:80"), Value: []byte("10.0.0.1:8080")},
		{Key: []byte("/services/api/host:api:9000"), Value: []byte("[2001:db8::1]:9000")},
	}}
	svcs, err := adapter(c).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastGet != "/services/" {
		t.Errorf("unexpected get prefix: %s", c.lastGet)
	}
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %d", len(svcs))
	}
	if svcs[0].Name != "web" || svcs[0].ID != "host:web:80" || svcs[0].IP != "10.0.0.1" || svcs[0].Port != 8080 {
		t.Errorf("unexpected first service: %+v", svcs[0])
	}
	if svcs[1].Name != "api" || svcs[1].ID != "host:api:9000" || svcs[1].IP != "2001:db8::1" || svcs[1].Port != 9000 {
		t.Errorf("unexpected second service: %+v", svcs[1])
	}
}

func TestServices_SkipsMalformedEntries(t *testing.T) {
	c := &fakeEtcdClient{kvs: []*mvccpb.KeyValue{
		{Key: []byte("/services/web/host:web:80"), Value: []byte("10.0.0.1:8080")},
		{Key: []byte("/services/missing-id"), Value: []byte("10.0.0.2:8080")},
		{Key: []byte("/services/api/host:api:9000"), Value: []byte("not-an-address")},
		{Key: []byte("/services/db/host:db:5432"), Value: []byte("10.0.0.3:not-a-port")},
	}}
	svcs, err := adapter(c).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service, got %d", len(svcs))
	}
	if svcs[0].Name != "web" {
		t.Errorf("unexpected service: %+v", svcs[0])
	}
}

func TestServices_ReturnsErrorOnGetFail(t *testing.T) {
	_, err := adapter(&fakeEtcdClient{getErr: errors.New("etcd: unavailable")}).Services()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPing_Success(t *testing.T) {
	if err := adapter(&fakeEtcdClient{}).Ping(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPing_Error(t *testing.T) {
	c := &fakeEtcdClient{pingErr: errors.New("etcd: unavailable")}
	if err := adapter(c).Ping(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNew_NoTLS(t *testing.T) {
	uri, _ := url.Parse("etcd://localhost:12379/services")
	a, err := (&Factory{}).New(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("expected adapter, got nil")
	}
}

func TestNew_TLSKeyPairError(t *testing.T) {
	t.Setenv("ETCD_CERT_FILE", "/nonexistent/cert.pem")
	t.Setenv("ETCD_KEY_FILE", "/nonexistent/key.pem")
	uri, _ := url.Parse("etcd://localhost:12379/services")
	_, err := (&Factory{}).New(uri)
	if err == nil {
		t.Fatal("expected error for missing TLS files, got nil")
	}
}
