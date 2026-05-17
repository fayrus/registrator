package main

import (
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/fayrus/registrator/internal/bridge"
	testassert "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flakyFactory struct {
	mu               sync.Mutex
	constructorFails int
	pingFails        int
	newCalls         int
	pingCalls        int
}

func (f *flakyFactory) New(uri *url.URL) (bridge.RegistryAdapter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.newCalls++
	if f.constructorFails > 0 {
		f.constructorFails--
		return nil, errors.New("transient constructor failure")
	}

	return &flakyAdapter{factory: f}, nil
}

type flakyAdapter struct {
	factory *flakyFactory
}

func (a *flakyAdapter) Ping() error {
	a.factory.mu.Lock()
	defer a.factory.mu.Unlock()

	a.factory.pingCalls++
	if a.factory.pingFails > 0 {
		a.factory.pingFails--
		return errors.New("transient ping failure")
	}

	return nil
}

func (a *flakyAdapter) Register(service *bridge.Service) error {
	return nil
}

func (a *flakyAdapter) Deregister(service *bridge.Service) error {
	return nil
}

func (a *flakyAdapter) Refresh(service *bridge.Service) error {
	return nil
}

func (a *flakyAdapter) Services() ([]*bridge.Service, error) {
	return nil, nil
}

func TestConnectWithRetryRetriesConstructorFailures(t *testing.T) {
	factory := &flakyFactory{constructorFails: 2}
	registered := bridge.AdapterFactories.Register(factory, "flaky-constructor")
	require.True(t, registered)
	t.Cleanup(func() {
		bridge.AdapterFactories.Unregister("flaky-constructor")
	})

	b, err := connectWithRetry(nil, "flaky-constructor://", bridge.Config{}, 2, 0)
	require.NoError(t, err)
	require.NotNil(t, b)

	testassert.Equal(t, 3, factory.newCalls)
	testassert.Equal(t, 1, factory.pingCalls)
}

func TestConnectWithRetryRetriesPingFailures(t *testing.T) {
	factory := &flakyFactory{pingFails: 2}
	registered := bridge.AdapterFactories.Register(factory, "flaky-ping")
	require.True(t, registered)
	t.Cleanup(func() {
		bridge.AdapterFactories.Unregister("flaky-ping")
	})

	b, err := connectWithRetry(nil, "flaky-ping://", bridge.Config{}, 2, 0*time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, b)

	testassert.Equal(t, 3, factory.newCalls)
	testassert.Equal(t, 3, factory.pingCalls)
}
