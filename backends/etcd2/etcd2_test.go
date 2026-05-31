package etcd2

import (
	"context"
	"errors"
	"testing"

	"github.com/fayrus/registrator/internal/bridge"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type fakeEtcd2Client struct {
	putErr    error
	delErr    error
	grantErr  error
	pingErr   error
	lastKey   string
	lastValue string
	lastTTL   int64
}

func (f *fakeEtcd2Client) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	f.lastKey = key
	f.lastValue = val
	return nil, f.putErr
}

func (f *fakeEtcd2Client) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	f.lastKey = key
	return nil, f.delErr
}

func (f *fakeEtcd2Client) Grant(ctx context.Context, ttl int64) (*clientv3.LeaseGrantResponse, error) {
	f.lastTTL = ttl
	if f.grantErr != nil {
		return nil, f.grantErr
	}
	return &clientv3.LeaseGrantResponse{ID: 1}, nil
}

func (f *fakeEtcd2Client) Status(ctx context.Context, endpoint string) (*clientv3.StatusResponse, error) {
	return nil, f.pingErr
}

func (f *fakeEtcd2Client) Endpoints() []string {
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

func adapter(c etcd2Client) *Etcd2Adapter {
	return &Etcd2Adapter{client: c, path: "/services"}
}

func TestRegister_BuildsCorrectKeyAndValue(t *testing.T) {
	c := &fakeEtcd2Client{}
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
	c := &fakeEtcd2Client{putErr: errors.New("etcd: connection refused")}
	if err := adapter(c).Register(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRegister_UsesLeaseWhenTTLSet(t *testing.T) {
	c := &fakeEtcd2Client{}
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
	c := &fakeEtcd2Client{grantErr: errors.New("etcd: lease error")}
	svc := testService()
	svc.TTL = 30
	if err := adapter(c).Register(svc); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeregister_BuildsCorrectKey(t *testing.T) {
	c := &fakeEtcd2Client{}
	if err := adapter(c).Deregister(testService()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastKey != "/services/web/web-1" {
		t.Errorf("unexpected key: %s", c.lastKey)
	}
}

func TestDeregister_ReturnsErrorOnDeleteFail(t *testing.T) {
	c := &fakeEtcd2Client{delErr: errors.New("etcd: connection refused")}
	if err := adapter(c).Deregister(testService()); err == nil {
		t.Fatal("expected error, got nil")
	}
}
