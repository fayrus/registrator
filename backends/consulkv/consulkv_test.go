package consul

import (
	"errors"
	"net/http"
	"net/http/httptest"
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

// newFromServer builds the adapter through the factory, as Registrator does,
// pointing it at a fake Consul HTTP API.
func newFromServer(t *testing.T, handler http.HandlerFunc) bridge.RegistryAdapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	u, err := url.Parse("consulkv://" + srv.Listener.Addr().String() + "/services")
	if err != nil {
		t.Fatal(err)
	}
	a, err := new(Factory).New(u)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{name: "succeeds when Consul reports a leader", status: http.StatusOK, body: `"10.0.0.1:8300"`},
		{name: "fails when Consul answers with an error", status: http.StatusInternalServerError, body: "no cluster leader", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			a := newFromServer(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			err := a.Ping()

			if tt.wantErr && err == nil {
				t.Fatal("expected Ping to fail")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Ping: %v", err)
			}
			if gotPath != "/v1/status/leader" {
				t.Fatalf("Ping queried %q, want /v1/status/leader", gotPath)
			}
		})
	}
}

func TestPing_FailsWhenConsulIsUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.Listener.Addr().String()
	srv.Close() // nothing listens on addr any more

	u, _ := url.Parse("consulkv://" + addr + "/services")
	a, err := new(Factory).New(u)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := a.Ping(); err == nil {
		t.Fatal("expected Ping to fail when Consul is unreachable")
	}
}
