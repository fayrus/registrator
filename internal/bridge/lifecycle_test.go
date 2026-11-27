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
	mu            sync.Mutex
	registered    []string
	deregistered  []string
	refreshed     []string
	failRegister  map[string]bool
	failRefresh   map[string]bool
	failDereg     map[string]bool
	external      []*Service
	servicesErr   error
	registerCalls int // every Register call, successful or not
}

func newRecordingAdapter() *recordingAdapter {
	return &recordingAdapter{failRegister: map[string]bool{}, failRefresh: map[string]bool{}, failDereg: map[string]bool{}}
}

func (a *recordingAdapter) Ping() error { return nil }

func (a *recordingAdapter) Register(s *Service) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.registerCalls++
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

func (a *recordingAdapter) setFailDeregister(id string, fail bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failDereg[id] = fail
}

func (a *recordingAdapter) setFailRegister(id string, fail bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.failRegister[id] = fail
}

// fakeClock lets tests move time forward past retry backoffs.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func useFakeClock(b *Bridge) *fakeClock {
	c := &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	b.Lock()
	b.now = c.Now
	b.Unlock()
	return c
}

func pendingKeys[V any](b *Bridge, m func() map[string]V) []string {
	b.Lock()
	defer b.Unlock()
	ids := make([]string, 0)
	for id := range m() {
		ids = append(ids, id)
	}
	return ids
}

func (b *Bridge) pendingIDs() []string {
	return pendingKeys(b, func() map[string]*pendingOp { return b.pendingDeregistrations })
}

func (b *Bridge) pendingRegistrationIDs() []string {
	return pendingKeys(b, func() map[string]*pendingOp { return b.pendingRegistrations })
}

// stopWithFailedDeregistration registers c1, makes its deregistration fail,
// and stops it, leaving one pending deregistration.
func stopWithFailedDeregistration(t *testing.T, config Config) (*Bridge, *fakeDocker, *recordingAdapter, *dockerapi.Container, *fakeClock) {
	t.Helper()
	b, docker, adapter := newLifecycleBridge(t, config)
	clock := useFakeClock(b)
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")
	adapter.setFailDeregister(Hostname+":web:80", true)
	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 0} })
	b.RemoveOnExit("c1")
	require.Equal(t, []string{Hostname + ":web:80"}, b.pendingIDs())
	return b, docker, adapter, c, clock
}

func TestPendingDeregistration_RetriedUntilItSucceeds(t *testing.T) {
	retries := map[string]func(b *Bridge, docker *fakeDocker){
		"on refresh":        func(b *Bridge, _ *fakeDocker) { b.Refresh() },
		"on sync":           func(b *Bridge, _ *fakeDocker) { b.Sync(true) },
		"on the retry tick": func(b *Bridge, _ *fakeDocker) { b.RetryPending() },
		"on the next event": func(b *Bridge, docker *fakeDocker) {
			docker.put(webContainer("c2", "api", "8080/tcp"))
			b.Add("c2")
		},
	}
	for name, retry := range retries {
		t.Run(name, func(t *testing.T) {
			b, docker, adapter, _, clock := stopWithFailedDeregistration(t, Config{DeregisterCheck: "always"})

			clock.advance(maxRetryDelay)
			retry(b, docker) // backend still failing: stays pending
			assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingIDs())

			adapter.setFailDeregister(Hostname+":web:80", false)
			clock.advance(maxRetryDelay)
			retry(b, docker)

			_, deregistered, _ := adapter.snapshot()
			assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
			assert.Empty(t, b.pendingIDs())
		})
	}
}

func TestPendingRetry_WaitsForBackoff(t *testing.T) {
	b, _, adapter, _, clock := stopWithFailedDeregistration(t, Config{DeregisterCheck: "always"})
	adapter.setFailDeregister(Hostname+":web:80", false)

	b.RetryPending() // not due yet: the first retry waits initialRetryDelay
	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered, "a burst of events right after a failure does not call the backend again")

	clock.advance(initialRetryDelay)
	b.RetryPending()
	_, deregistered, _ = adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
}

func TestRetryDelay_DoublesUpToMax(t *testing.T) {
	assert.Equal(t, initialRetryDelay, retryDelay(1))
	assert.Equal(t, 2*initialRetryDelay, retryDelay(2))
	assert.Equal(t, 4*initialRetryDelay, retryDelay(3))
	assert.Equal(t, maxRetryDelay, retryDelay(10))
	assert.Equal(t, maxRetryDelay, retryDelay(1000), "no overflow for long outages")
}

func TestPendingDeregistration_DroppedWhenServiceIsRegisteredAgain(t *testing.T) {
	b, docker, adapter, c, clock := stopWithFailedDeregistration(t, Config{DeregisterCheck: "always"})

	// The container comes back before the backend recovers.
	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{Running: true} })
	b.Add("c1")
	assert.Empty(t, b.pendingIDs(), "a successful registration cancels the pending deregistration")

	adapter.setFailDeregister(Hostname+":web:80", false)
	clock.advance(maxRetryDelay)
	b.RetryPending()

	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered, "the running service is not deregistered by a stale retry")
	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
}

func TestPendingDeregistration_GivesUpAfterMaxAttempts(t *testing.T) {
	b, _, adapter, _, clock := stopWithFailedDeregistration(t, Config{DeregisterCheck: "always"})

	for i := 1; i < maxDeregisterAttempts-1; i++ {
		clock.advance(maxRetryDelay)
		b.RetryPending()
	}
	assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingIDs(), "still pending before the last attempt")

	clock.advance(maxRetryDelay)
	b.RetryPending()
	assert.Empty(t, b.pendingIDs(), "dropped after the last failed attempt")

	adapter.setFailDeregister(Hostname+":web:80", false)
	clock.advance(maxRetryDelay)
	b.RetryPending()
	_, deregistered, _ := adapter.snapshot()
	assert.Empty(t, deregistered)
}

func TestPendingDeregistration_IncludesDeadContainerServices(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "on-success", RefreshTtl: 30, RefreshInterval: 10})
	clock := useFakeClock(b)
	c := webContainer("c1", "web", "80/tcp")
	docker.put(c)
	b.Add("c1")
	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 1} })
	b.RemoveOnExit("c1")
	require.Contains(t, b.deadContainers, "c1")

	adapter.setFailDeregister(Hostname+":web:80", true)
	b.Remove("c1")
	assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingIDs())

	adapter.setFailDeregister(Hostname+":web:80", false)
	clock.advance(maxRetryDelay)
	b.RetryPending()
	_, deregistered, _ := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:80"}, deregistered)
}

func TestPendingRegistration_PartialFailureIsRetried(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	clock := useFakeClock(b)
	docker.put(webContainer("c1", "web", "80/tcp", "443/tcp"))
	adapter.setFailRegister(Hostname+":web:443", true)

	b.Add("c1")
	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	assert.Equal(t, []string{Hostname + ":web:443"}, b.pendingRegistrationIDs())

	adapter.setFailRegister(Hostname+":web:443", false)
	clock.advance(initialRetryDelay)
	b.RetryPending()

	assert.ElementsMatch(t, []string{Hostname + ":web:80", Hostname + ":web:443"}, serviceIDs(b.services["c1"]))
	assert.Empty(t, b.pendingRegistrationIDs())

	b.Refresh()
	_, _, refreshed := adapter.snapshot()
	assert.ElementsMatch(t, []string{Hostname + ":web:80", Hostname + ":web:443"}, refreshed, "a service registered on retry is refreshed like the others")
}

func TestPendingRegistration_TotalFailureIsRetriedWithoutResync(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	clock := useFakeClock(b)
	docker.put(webContainer("c1", "web", "80/tcp"))
	adapter.setFailRegister(Hostname+":web:80", true)

	b.Add("c1")
	assert.NotContains(t, b.services, "c1")
	assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingRegistrationIDs())

	adapter.setFailRegister(Hostname+":web:80", false)
	clock.advance(initialRetryDelay)
	b.RetryPending() // the 30-second ticker, with -resync=0

	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	assert.Empty(t, b.pendingRegistrationIDs())
}

func TestPendingRegistration_NoAttemptLimit(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	clock := useFakeClock(b)
	docker.put(webContainer("c1", "web", "80/tcp"))
	adapter.setFailRegister(Hostname+":web:80", true)
	b.Add("c1")

	for i := 0; i < 3*maxDeregisterAttempts; i++ {
		clock.advance(maxRetryDelay)
		b.RetryPending()
	}
	assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingRegistrationIDs(), "a running container keeps retrying")

	adapter.setFailRegister(Hostname+":web:80", false)
	clock.advance(maxRetryDelay)
	b.RetryPending()
	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
}

func TestPendingRegistration_DroppedWhenContainerStops(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{DeregisterCheck: "always"})
	clock := useFakeClock(b)
	c := webContainer("c1", "web", "80/tcp", "443/tcp")
	docker.put(c)
	adapter.setFailRegister(Hostname+":web:443", true)
	b.Add("c1")
	require.Equal(t, []string{Hostname + ":web:443"}, b.pendingRegistrationIDs())

	docker.set(func(f *fakeDocker) { c.State = dockerapi.State{ExitCode: 0} })
	b.RemoveOnExit("c1")
	assert.Empty(t, b.pendingRegistrationIDs())

	adapter.setFailRegister(Hostname+":web:443", false)
	clock.advance(maxRetryDelay)
	b.RetryPending()
	registered, _, _ := adapter.snapshot()
	assert.Equal(t, []string{Hostname + ":web:80"}, registered, "a stopped container is not registered by a late retry")
}

func (a *recordingAdapter) registerCallCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.registerCalls
}

// A resync re-runs add for a container whose services all failed. It must
// leave the pending registration to the retry queue instead of calling the
// backend on every tick and restarting the backoff.
func TestPendingRegistration_ResyncHonorsBackoff(t *testing.T) {
	b, docker, adapter := newLifecycleBridge(t, Config{})
	clock := useFakeClock(b)
	docker.put(webContainer("c1", "web", "80/tcp"))
	docker.set(func(f *fakeDocker) { f.listed = []string{"c1"} })
	adapter.setFailRegister(Hostname+":web:80", true)

	b.Add("c1")
	require.Equal(t, 1, adapter.registerCallCount())

	for i := 0; i < 3; i++ {
		clock.advance(initialRetryDelay / 4)
		b.Sync(true)
	}
	assert.Equal(t, 1, adapter.registerCallCount(), "resync before the backoff elapses does not call the backend")
	b.Lock()
	assert.Equal(t, 1, b.pendingRegistrations[Hostname+":web:80"].attempts, "the backoff is not restarted")
	b.Unlock()

	adapter.setFailRegister(Hostname+":web:80", false)
	clock.advance(initialRetryDelay)
	b.Sync(true)
	assert.Equal(t, []string{Hostname + ":web:80"}, serviceIDs(b.services["c1"]))
	assert.Empty(t, b.pendingRegistrationIDs())
}

// A container whose services all failed to register can stop without its die
// event being processed (missed event). A later retry must not register it.
func TestPendingRegistration_VerifiesContainerBeforeRetry(t *testing.T) {
	tests := []struct {
		name        string
		stop        func(f *fakeDocker, c *dockerapi.Container)
		wantPending bool
	}{
		{
			name: "drops it when the container stopped without a die event",
			stop: func(f *fakeDocker, c *dockerapi.Container) { c.State = dockerapi.State{ExitCode: 0} },
		},
		{
			name: "drops it when the container was removed",
			stop: func(f *fakeDocker, c *dockerapi.Container) { delete(f.containers, c.ID) },
		},
		{
			name:        "keeps it when the container cannot be inspected",
			stop:        func(f *fakeDocker, c *dockerapi.Container) { f.inspectFail[c.ID] = true },
			wantPending: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, docker, adapter := newLifecycleBridge(t, Config{})
			clock := useFakeClock(b)
			c := webContainer("c1", "web", "80/tcp")
			docker.put(c)
			adapter.setFailRegister(Hostname+":web:80", true)
			b.Add("c1")
			require.Equal(t, []string{Hostname + ":web:80"}, b.pendingRegistrationIDs())

			docker.set(func(f *fakeDocker) { tt.stop(f, c) })
			adapter.setFailRegister(Hostname+":web:80", false)
			calls := adapter.registerCallCount()
			clock.advance(initialRetryDelay)
			b.RetryPending()

			assert.Equal(t, calls, adapter.registerCallCount(), "the backend is not called for a container that is not running")
			assert.NotContains(t, b.services, "c1")
			if tt.wantPending {
				assert.Equal(t, []string{Hostname + ":web:80"}, b.pendingRegistrationIDs())
			} else {
				assert.Empty(t, b.pendingRegistrationIDs())
			}
		})
	}
}
