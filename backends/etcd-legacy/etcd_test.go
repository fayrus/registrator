package etcdlegacy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	etcd2 "github.com/coreos/go-etcd/etcd"
	etcd "gopkg.in/coreos/go-etcd.v0/etcd"
)

type fakeV0Client struct {
	getErr  error
	node    *etcd.Node
	lastGet string
}

func (f *fakeV0Client) Set(key string, value string, ttl uint64) (*etcd.Response, error) {
	return nil, nil
}

func (f *fakeV0Client) Delete(key string, recursive bool) (*etcd.Response, error) {
	return nil, nil
}

func (f *fakeV0Client) Get(key string, sort, recursive bool) (*etcd.Response, error) {
	f.lastGet = key
	return &etcd.Response{Node: f.node}, f.getErr
}

func (f *fakeV0Client) SendRequest(rr *etcd.RawRequest) (*etcd.RawResponse, error) {
	return nil, nil
}

func (f *fakeV0Client) SyncCluster() bool {
	return true
}

type fakeV2Client struct {
	getErr  error
	node    *etcd2.Node
	lastGet string
}

func (f *fakeV2Client) Set(key string, value string, ttl uint64) (*etcd2.Response, error) {
	return nil, nil
}

func (f *fakeV2Client) Delete(key string, recursive bool) (*etcd2.Response, error) {
	return nil, nil
}

func (f *fakeV2Client) Get(key string, sort, recursive bool) (*etcd2.Response, error) {
	f.lastGet = key
	return &etcd2.Response{Node: f.node}, f.getErr
}

func (f *fakeV2Client) SendRequest(rr *etcd2.RawRequest) (*etcd2.RawResponse, error) {
	return nil, nil
}

func (f *fakeV2Client) SyncCluster() bool {
	return true
}

func TestServices_ListsRegisteredServicesFromV2Client(t *testing.T) {
	c := &fakeV2Client{node: &etcd2.Node{
		Key: "/services",
		Dir: true,
		Nodes: etcd2.Nodes{
			{
				Key: "/services/web",
				Dir: true,
				Nodes: etcd2.Nodes{
					{Key: "/services/web/host:web:80", Value: "10.0.0.1:8080"},
				},
			},
			{
				Key: "/services/api",
				Dir: true,
				Nodes: etcd2.Nodes{
					{Key: "/services/api/host:api:9000", Value: "[2001:db8::1]:9000"},
				},
			},
		},
	}}

	svcs, err := (&EtcdAdapter{client2: c, path: "/services"}).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastGet != "/services" {
		t.Errorf("unexpected get path: %s", c.lastGet)
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

func TestServices_ListsRegisteredServicesFromV0Client(t *testing.T) {
	c := &fakeV0Client{node: &etcd.Node{
		Key: "/services",
		Dir: true,
		Nodes: etcd.Nodes{
			{
				Key: "/services/web",
				Dir: true,
				Nodes: etcd.Nodes{
					{Key: "/services/web/host:web:80", Value: "10.0.0.1:8080"},
				},
			},
		},
	}}

	svcs, err := (&EtcdAdapter{client: c, path: "/services"}).Services()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.lastGet != "/services" {
		t.Errorf("unexpected get path: %s", c.lastGet)
	}
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service, got %d", len(svcs))
	}
	if svcs[0].Name != "web" || svcs[0].ID != "host:web:80" || svcs[0].IP != "10.0.0.1" || svcs[0].Port != 8080 {
		t.Errorf("unexpected service: %+v", svcs[0])
	}
}

func TestServices_SkipsMalformedEntries(t *testing.T) {
	c := &fakeV2Client{node: &etcd2.Node{
		Key: "/services",
		Dir: true,
		Nodes: etcd2.Nodes{
			{Key: "/services/web/host:web:80", Value: "10.0.0.1:8080"},
			{Key: "/services/missing-id", Value: "10.0.0.2:8080"},
			{Key: "/services/api/host:api:9000", Value: "not-an-address"},
			{Key: "/services/db/host:db:5432", Value: "10.0.0.3:not-a-port"},
		},
	}}

	svcs, err := (&EtcdAdapter{client2: c, path: "/services"}).Services()
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

func TestFetchVersion_TimesOutOnSlowEndpoint(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	_, err := fetchVersion(&http.Client{Timeout: 50 * time.Millisecond}, srv.URL)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestNew_SelectsClientFromVersion(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantV0  bool
	}{
		{name: "v0.4", version: "etcd 0.4.6", wantV0: true},
		{name: "v2", version: `{"etcdserver":"3.5.17","etcdcluster":"3.5.0"}`, wantV0: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.version))
			}))
			defer srv.Close()

			uri, err := url.Parse("etcd-legacy://" + srv.Listener.Addr().String() + "/services")
			if err != nil {
				t.Fatal(err)
			}
			adapter, err := new(Factory).New(uri)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			a := adapter.(*EtcdAdapter)
			if gotV0 := a.client != nil; gotV0 != tc.wantV0 {
				t.Errorf("v0 client selected = %v, want %v", gotV0, tc.wantV0)
			}
		})
	}
}

func TestNew_ReturnsErrorWhenVersionUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	addr := srv.Listener.Addr().String()
	srv.Close()

	uri, err := url.Parse("etcd-legacy://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := new(Factory).New(uri); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestServices_ReturnsErrorOnGetFail(t *testing.T) {
	_, err := (&EtcdAdapter{
		client2: &fakeV2Client{getErr: errors.New("etcd: unavailable")},
		path:    "/services",
	}).Services()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
