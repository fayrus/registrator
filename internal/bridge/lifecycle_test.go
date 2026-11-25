package bridge

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeDocker serves the subset of the Docker API the bridge uses:
// container inspect and container list (with and without a status filter).
type fakeDocker struct {
	mu          sync.Mutex
	containers  map[string]*dockerapi.Container
	listed      []string // IDs returned by an unfiltered list
	nonExited   []string // IDs returned by a list with a status filter
	inspectFail map[string]bool
	listFail    bool
	onInspect   func(id string) // called before an inspect response, without the fake's lock
}

func newFakeDocker(t *testing.T) (*fakeDocker, *dockerapi.Client) {
	t.Helper()
	f := &fakeDocker{containers: map[string]*dockerapi.Container{}, inspectFail: map[string]bool{}}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	client, err := dockerapi.NewClient(srv.URL)
	require.NoError(t, err)
	return f, client
}

func (f *fakeDocker) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/containers/json"):
		if f.listFail {
			http.Error(w, "list failed", http.StatusInternalServerError)
			return
		}
		ids := f.listed
		if r.URL.Query().Get("filters") != "" {
			ids = f.nonExited
		}
		out := make([]dockerapi.APIContainers, 0, len(ids))
		for _, id := range ids {
			out = append(out, dockerapi.APIContainers{ID: id})
		}
		writeJSON(w, out)
	case strings.HasPrefix(path, "/containers/") && strings.HasSuffix(path, "/json"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/containers/"), "/json")
		if hook := f.onInspect; hook != nil {
			f.mu.Unlock()
			hook(id)
			f.mu.Lock()
		}
		if f.inspectFail[id] {
			http.Error(w, "inspect failed", http.StatusInternalServerError)
			return
		}
		c, ok := f.containers[id]
		if !ok {
			http.Error(w, "no such container", http.StatusNotFound)
			return
		}
		writeJSON(w, c)
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fakeDocker) put(c *dockerapi.Container) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.containers[c.ID] = c
}

func (f *fakeDocker) set(fn func(f *fakeDocker)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

// recordingAdapter records registry calls and can fail selected service IDs.
type recordingAdapter struct {
	mu           sync.Mutex
	registered   []string
	deregistered []string
	refreshed    []string
	failRegister map[string]bool
	failRefresh  map[string]bool
	failDereg    map[string]bool
	external     []*Service
	servicesErr  error
}

func newRecordingAdapter() *recordingAdapter {
	return &recordingAdapter{failRegister: map[string]bool{}, failRefresh: map[string]bool{}, failDereg: map[string]bool{}}
}

func (a *recordingAdapter) Ping() error { return nil }

func (a *recordingAdapter) Register(s *Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failRegister[s.ID] {
		return errors.New("register failed")
	}
	a.registered = append(a.registered, s.ID)
	return nil
}

func (a *recordingAdapter) Deregister(s *Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failDereg[s.ID] {
		return errors.New("deregister failed")
	}
	a.deregistered = append(a.deregistered, s.ID)
	return nil
}

func (a *recordingAdapter) Refresh(s *Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.failRefresh[s.ID] {
		return errors.New("refresh failed")
	}
	a.refreshed = append(a.refreshed, s.ID)
	return nil
}

func (a *recordingAdapter) Services() ([]*Service, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.external, a.servicesErr
}

func (a *recordingAdapter) snapshot() (registered, deregistered, refreshed []string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.registered...), append([]string(nil), a.deregistered...), append([]string(nil), a.refreshed...)
}

const testHostIP = "10.0.0.1"

func newLifecycleBridge(t *testing.T, config Config) (*Bridge, *fakeDocker, *recordingAdapter) {
	t.Helper()
	docker, client := newFakeDocker(t)
	adapter := newRecordingAdapter()
	b := &Bridge{
		docker:         client,
		registry:       adapter,
		config:         config,
		services:       make(map[string][]*Service),
		deadContainers: make(map[string]*DeadContainer),
	}
	return b, docker, adapter
}

// webContainer returns a running container that publishes the given
// container ports (e.g. "80/tcp") on host ports starting at 30000. Ports are
// bound to testHostIP rather than 0.0.0.0, so the bridge does not resolve the
// machine hostname through DNS for every service.
func webContainer(id, name string, ports ...string) *dockerapi.Container {
	bindings := map[dockerapi.Port][]dockerapi.PortBinding{}
	for i, p := range ports {
		bindings[dockerapi.Port(p)] = []dockerapi.PortBinding{{HostIP: testHostIP, HostPort: strconv.Itoa(30000 + i)}}
	}
	return &dockerapi.Container{
		ID:   id,
		Name: "/" + name,
		Config: &dockerapi.Config{
			Image: "example/" + name + ":1.0",
		},
		HostConfig:      &dockerapi.HostConfig{NetworkMode: "bridge"},
		NetworkSettings: &dockerapi.NetworkSettings{IPAddress: "172.17.0.2", Ports: bindings},
		State:           dockerapi.State{Running: true},
	}
}

func serviceIDs(services []*Service) []string {
	ids := make([]string, 0, len(services))
	for _, s := range services {
		ids = append(ids, s.ID)
	}
	return ids
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name           string
		container      *dockerapi.Container
		inspectFails   bool
		failRegister   string
		wantRegistered []string
	}{
		{
			name:           "registers one service per published port",
			container:      webContainer("c1", "web", "80/tcp", "443/tcp"),
			wantRegistered: []string{Hostname + ":web:80", Hostname + ":web:443"},
		},
		{
			name:      "ignores a container without published ports",
			container: webContainer("c1", "web"),
		},
		{
			name: "ignores a port that is exposed but not published",
			container: func() *dockerapi.Container {
				c := webContainer("c1", "web")
				c.NetworkSettings.Ports[dockerapi.Port("80/tcp")] = nil
				return c
			}(),
		},
		{
			name:         "ignores a container that cannot be inspected",
			container:    webContainer("c1", "web", "80/tcp"),
			inspectFails: true,
		},
		{
			name:           "does not keep a service whose registration fails",
			container:      webContainer("c1", "web", "80/tcp", "443/tcp"),
			failRegister:   Hostname + ":web:443",
			wantRegistered: []string{Hostname + ":web:80"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, docker, adapter := newLifecycleBridge(t, Config{})
			docker.put(tt.container)
			docker.set(func(f *fakeDocker) { f.inspectFail[tt.container.ID] = tt.inspectFails })
			if tt.failRegister != "" {
				adapter.failRegister[tt.failRegister] = true
			}

			b.Add(tt.container.ID)

			registered, _, _ := adapter.snapshot()
			assert.ElementsMatch(t, tt.wantRegistered, registered)
			assert.ElementsMatch(t, tt.wantRegistered, serviceIDs(b.services[tt.container.ID]))
		})
	}
}

func TestAdd_SkipsAlreadyRegisteredContainer(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.put(webContainer("c1", "web", "80/tcp"))

	b.Add("c1")
	b.Add("c1")

	registered, _, _ := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:80"}, registered)
}

func TestRemoveOnExit(t *testing.T) {
	tests := []struct {
		name            string
		deregisterCheck string
		state           *dockerapi.State // nil: container no longer exists
		inspectFails    bool
		wantDeregister  bool
		wantTracked     bool // services stay tracked as live (container still running)
	}{
		{name: "always deregisters regardless of exit code", deregisterCheck: "always", state: &dockerapi.State{ExitCode: 1}, wantDeregister: true},
		{name: "always deregisters a container removed from Docker", deregisterCheck: "always", state: nil, wantDeregister: true},
		{name: "always deregisters when inspect fails", deregisterCheck: "always", state: &dockerapi.State{}, inspectFails: true, wantDeregister: true},
		{name: "always keeps a container that is running again", deregisterCheck: "always", state: &dockerapi.State{Running: true}, wantTracked: true},
		{name: "always keeps a container that is restarting", deregisterCheck: "always", state: &dockerapi.State{Restarting: true, ExitCode: 1}, wantTracked: true},
		{name: "on-success deregisters on exit code 0", deregisterCheck: "on-success", state: &dockerapi.State{ExitCode: 0}, wantDeregister: true},
		{name: "on-success deregisters when stopped by a signal", deregisterCheck: "on-success", state: &dockerapi.State{ExitCode: 128 + 15}, wantDeregister: true},
		{name: "on-success deregisters a container removed from Docker", deregisterCheck: "on-success", state: nil, wantDeregister: true},
		{name: "on-success keeps a container that failed", deregisterCheck: "on-success", state: &dockerapi.State{ExitCode: 1}},
		{name: "on-success keeps a container that is still running", deregisterCheck: "on-success", state: &dockerapi.State{Running: true}, wantTracked: true},
		{name: "on-success keeps a container when inspect fails", deregisterCheck: "on-success", state: &dockerapi.State{}, inspectFails: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: tt.deregisterCheck})
			c := webContainer("c1", "web", "80/tcp")
			docker.put(c)
			b.Add("c1")
			require.Len(t, b.services["c1"], 1)

			docker.set(func(f *fakeDocker) {
				if tt.state == nil {
					delete(f.containers, "c1")
					return
				}
				c.State = *tt.state
				f.inspectFail["c1"] = tt.inspectFails
			})

			b.RemoveOnExit("c1")

			_, deregistered, _ := adapter.snapshot()
			if tt.wantDeregister {
				assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
			} else {
				assert.Empty(t, deregistered)
			}
			if tt.wantTracked {
				assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]), "a running container keeps its services tracked and refreshed")
			} else {
				assert.NotContains(t, b.services, "c1")
			}
		})
	}
}

// A restart emits die and start milliseconds apart, and each event is handled
// in its own goroutine. Both orders must leave the running container registered.
func TestRestart_EventOrder(t *testing.T) {
	t.Run("start handled before die", func(t *testing.T) {
		b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "always"})
		docker.put(webContainer("c1", "web", "80/tcp"))
		b.Add("c1")

		b.Add("c1")          // start: services already present, nothing to do
		b.RemoveOnExit("c1") // die: the container is running again

		_, deregistered, _ := adapter.snapshot()
		assert.Empty(t, deregistered)
		assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	})

	t.Run("die handled before start", func(t *testing.T) {
		b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "always"})
		c := webContainer("c1", "web", "80/tcp")
		docker.put(c)
		b.Add("c1")

		docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 0} })
		b.RemoveOnExit("c1") // die: the container has stopped
		docker.set(func(f *fakeDocker) { c.State = dockerapi.State{Running: true} })
		b.Add("c1") // start: registers it again

		registered, deregistered, _ := adapter.snapshot()
		assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
		assert.Equal(t, []string{Hostname + ":web:80", Hostname + ":web:80"}, registered)
		assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	})
}

// The die handler must decide and remove atomically: an Add for the same
// container that arrives while the die handler inspects the container has to
// wait, instead of slipping in between the decision and the removal.
func TestRemoveOnExit_InspectsUnderBridgeLock(t *testing.T) {
	b, docker, _ := newLifecycleBridge(t, Config{DeregisterCheck: "on-success"})
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")
	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 0} })

	lockedDuringInspect := make(chan bool, 1)
	docker.set(func(f *fakeDocker) {
		f.onInspect = func(id string) {
			if id != "c1" {
				return
			}
			acquired := b.TryLock()
			if acquired {
				b.Unlock()
			}
			select {
			case lockedDuringInspect <- !acquired:
			default:
			}
		}
	})

	b.RemoveOnExit("c1")

	assert.True(t, <-lockedDuringInspect, "the bridge lock is held while the exited container is inspected")
}

func TestRemove_DeregisterFailureDoesNotStopOtherServices(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.put(webContainer("c1", "web", "80/tcp", "443/tcp"))
	b.Add("c1")
	adapter.failDereg[Hostname+":web:80"] = true

	b.Remove("c1")

	_, deregistered, _ := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:443"}, deregistered)
	assert.NotContains(t, b.services, "c1")
}

func TestDeadContainer_TTLLifecycle(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "on-success", RefreshTtl: 30, RefreshInterval: 10})
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")

	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 1} })
	b.RemoveOnExit("c1")

	require.Contains(t, b.deadContainers, "c1", "a failed container is kept as dead while its TTL lasts")
	assert.Equal(t, 30, b.deadContainers["c1"].TTL)

	b.Refresh()
	assert.Equal(t, 20, b.deadContainers["c1"].TTL)
	_, _, refreshed := adapter.snapshot()
	assert.Empty(t, refreshed, "dead containers are not refreshed in the registry")

	b.Refresh()
	b.Refresh()
	assert.NotContains(t, b.deadContainers, "c1", "the dead container is forgotten once its TTL expires")

	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered)
}

func TestDeadContainer_RestartRestoresServices(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "on-success", RefreshTtl: 30, RefreshInterval: 10})
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")

	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 1} })
	b.RemoveOnExit("c1")
	require.Contains(t, b.deadContainers, "c1")

	b.Add("c1")

	assert.NotContains(t, b.deadContainers, "c1")
	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	registered, _, _ := adapter.snapshot()
	assert.Len(t, registered, 1, "restored services are not registered again")
}

func TestRemove_DeregistersDeadContainerServices(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "on-success", RefreshTtl: 30, RefreshInterval: 10})
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")
	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 1} })
	b.RemoveOnExit("c1")
	require.Contains(t, b.deadContainers, "c1")

	b.Remove("c1")

	_, deregistered, _ := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
	assert.NotContains(t, b.deadContainers, "c1")
}

func TestRefresh_ContinuesAfterFailure(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.put(webContainer("c1", "web", "80/tcp", "443/tcp"))
	b.Add("c1")
	adapter.failRefresh[Hostname+":web:80"] = true

	b.Refresh()

	_, _, refreshed := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:443"}, refreshed)
}

func TestSync(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.put(webContainer("c1", "web", "80/tcp"))
	docker.put(webContainer("c2", "api", "8080/tcp"))
	b.Add("c1")
	docker.set(func(f *fakeDocker) { f.listed = []string{"c1", "c2"} })

	b.Sync(false)

	registered, _, _ := adapter.snapshot()
	assert.ElementsMatch(t, []string{
		Hostname + ":web:80", // initial Add
		Hostname + ":web:80", // re-registered by Sync
		Hostname + ":api:8080",
	}, registered)
	assert.Contains(t, b.services, "c2")
}

func TestSync_ReRegisterFailureKeepsService(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.put(webContainer("c1", "web", "80/tcp"))
	b.Add("c1")
	docker.set(func(f *fakeDocker) { f.listed = []string{"c1"} })
	adapter.failRegister[Hostname+":web:80"] = true

	b.Sync(false)

	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]), "a failed re-registration does not drop the tracked service")
}

func TestSync_QuietSkipsWhenListingFails(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	docker.set(func(f *fakeDocker) { f.listFail = true })

	b.Sync(true)

	registered, _, _ := adapter.snapshot()
	assert.Empty(t, registered)
}

func TestSync_CleanupRemovesStaleServices(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{Cleanup: true, DeregisterCheck: "always"})
	docker.put(webContainer("c1", "web", "80/tcp"))
	docker.put(webContainer("c2", "api", "8080/tcp"))
	b.Add("c1")
	b.Add("c2")
	// c2 is gone from Docker: not listed and not among the non-exited containers.
	docker.set(func(f *fakeDocker) {
		f.listed = []string{"c1"}
		f.nonExited = []string{"c1"}
		delete(f.containers, "c2")
	})

	b.Sync(true)

	assert.Eventually(t, func() bool {
		_, deregistered, _ := adapter.snapshot()
		return len(deregistered) == 1 && deregistered[0] == Hostname+":api:8080"
	}, 2*time.Second, 10*time.Millisecond)
	b.Lock()
	defer b.Unlock()
	assert.Contains(t, b.services, "c1")
	assert.NotContains(t, b.services, "c2")
}

func TestCleanup_SkipsWhenNonExitedListingFails(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{Cleanup: true, DeregisterCheck: "always"})
	docker.put(webContainer("c1", "web", "80/tcp"))
	b.Add("c1")
	adapter.external = []*Service{{ID: Hostname + ":gone:80", Name: "gone"}}
	docker.set(func(f *fakeDocker) { f.listFail = true })

	b.Lock()
	b.cleanupStaleAndDangling()
	b.Unlock()

	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered, "neither stale nor dangling services are removed without the container list")
	assert.Contains(t, b.services, "c1")
}

func TestCleanup_RemovesOnlyDanglingServicesOfThisHost(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{Cleanup: true})
	docker.put(webContainer("c1", "web", "80/tcp"))
	b.Add("c1")
	docker.set(func(f *fakeDocker) {
		f.listed = []string{"c1"}
		f.nonExited = []string{"c1"}
	})
	adapter.external = []*Service{
		{ID: Hostname + ":web:80", Name: "web"},           // registered by this bridge: kept
		{ID: Hostname + ":old:80", Name: "old"},           // this host, no container: dangling
		{ID: "other-host:old:80", Name: "old"},            // another host: never touched
		{ID: "not-a-registrator-id", Name: "unmanaged"},   // not created by Registrator
		{ID: Hostname + ":broken:443", Name: "broken"},    // dangling, deregistration fails
		{ID: Hostname + ":web:8080:udp", Name: "web-udp"}, // same container name, other service name: dangling
	}
	adapter.failDereg[Hostname+":broken:443"] = true

	b.Sync(true)

	_, deregistered, _ := adapter.snapshot()
	assert.ElementsMatch(t, []string{Hostname + ":old:80", Hostname + ":web:8080:udp"}, deregistered)
}

func TestCleanup_SkipsWhenRegistryListingFails(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{Cleanup: true})
	docker.set(func(f *fakeDocker) { f.nonExited = []string{} })
	adapter.servicesErr = errors.New("registry unavailable")

	b.Sync(true)

	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered)
}
