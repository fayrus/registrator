package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	dockerapi "github.com/fsouza/go-dockerclient"
	"github.com/fayrus/registrator/internal/bridge"
)

var Version string

var hostIp = flag.String("ip", "", "IP for ports mapped to the host")
var internal = flag.Bool("internal", false, "Use internal ports instead of published ones")
var explicit = flag.Bool("explicit", false, "Only register containers which have SERVICE_NAME label set")
var useIpFromLabel = flag.String("useIpFromLabel", "", "Use IP which is stored in a label assigned to the container")
var ipFromContainer = flag.Bool("ip-from-container", false, "Use container IP instead of host IP for service registration")
var refreshInterval = flag.Int("ttl-refresh", 0, "Frequency with which service TTLs are refreshed")
var refreshTtl = flag.Int("ttl", 0, "TTL for services (default is no expiry)")
var forceTags = flag.String("tags", "", "Append tags for all registered services (supports Go template)")
var resyncInterval = flag.Int("resync", 0, "Frequency with which services are resynchronized")
var deregister = flag.String("deregister", "always", "Deregister exited services \"always\" or \"on-success\"")
var retryAttempts = flag.Int("retry-attempts", 0, "Max retry attempts to establish a connection with the backend. Use -1 for infinite retries")
var retryInterval = flag.Int("retry-interval", 2000, "Interval (in millisecond) between retry-attempts.")
var cleanup = flag.Bool("cleanup", false, "Remove dangling services")

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func connectWithRetry(docker *dockerapi.Client, adapterURI string, config bridge.Config, retryAttempts int, retryInterval time.Duration) (*bridge.Bridge, error) {
	attempt := 0
	for retryAttempts == -1 || attempt <= retryAttempts {
		log.Printf("Connecting to backend (%v/%v)", attempt, retryAttempts)

		b, err := bridge.New(docker, adapterURI, config)
		if err == nil {
			err = b.Ping()
		}
		if err == nil {
			return b, nil
		}
		if retryAttempts != -1 && attempt == retryAttempts {
			return nil, err
		}

		time.Sleep(retryInterval)
		attempt++
	}

	return nil, errors.New("unreachable retry state")
}

func validateArgs() error {
	if flag.NArg() == 0 {
		fmt.Fprint(os.Stderr, "Missing required argument for registry URI.\n\n")
		flag.Usage()
		os.Exit(2)
	}
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "Extra unparsed arguments:")
		fmt.Fprintln(os.Stderr, " ", strings.Join(flag.Args()[1:], " "))
		fmt.Fprint(os.Stderr, "Options should come before the registry URI argument.\n\n")
		flag.Usage()
		os.Exit(2)
	}
	return nil
}

func validateFlags() error {
	if (*refreshTtl == 0 && *refreshInterval > 0) || (*refreshTtl > 0 && *refreshInterval == 0) {
		return errors.New("-ttl and -ttl-refresh must be specified together or not at all")
	}
	if *refreshTtl > 0 && *refreshTtl <= *refreshInterval {
		return errors.New("-ttl must be greater than -ttl-refresh")
	}
	if *retryInterval <= 0 {
		return errors.New("-retry-interval must be greater than 0")
	}
	if *deregister != "always" && *deregister != "on-success" {
		return errors.New("-deregister must be \"always\" or \"on-success\"")
	}
	return nil
}

func setupDockerHost() {
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	if runtime.GOOS != "windows" {
		_ = os.Setenv("DOCKER_HOST", "unix:///tmp/docker.sock")
	} else {
		_ = os.Setenv("DOCKER_HOST", "npipe:////./pipe/docker_engine")
	}
}

func startRefreshTicker(b *bridge.Bridge, interval int, quit chan struct{}) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				b.Refresh()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func startResyncTicker(b *bridge.Bridge, interval int, quit chan struct{}) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				b.Sync(true)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Println(Version)
		os.Exit(0)
	}
	log.Printf("Starting registrator %s ...", Version)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [options] <registry URI>\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	assert(validateArgs())

	if *hostIp != "" {
		log.Println("Forcing host IP to", *hostIp)
	}

	assert(validateFlags())
	setupDockerHost()

	docker, err := dockerapi.NewClientFromEnv()
	assert(err)

	b, err := connectWithRetry(docker, flag.Arg(0), bridge.Config{
		HostIp:          *hostIp,
		Internal:        *internal,
		Explicit:        *explicit,
		UseIpFromLabel:  *useIpFromLabel,
		IpFromContainer: *ipFromContainer,
		ForceTags:       *forceTags,
		RefreshTtl:      *refreshTtl,
		RefreshInterval: *refreshInterval,
		DeregisterCheck: *deregister,
		Cleanup:         *cleanup,
	}, *retryAttempts, time.Duration(*retryInterval)*time.Millisecond)
	assert(err)

	// Start event listener before listing containers to avoid missing anything
	events := make(chan *dockerapi.APIEvents)
	assert(docker.AddEventListener(events))
	log.Println("Listening for Docker events ...")

	b.Sync(false)

	quit := make(chan struct{})
	startRefreshTicker(b, *refreshInterval, quit)
	startResyncTicker(b, *resyncInterval, quit)

	// Process Docker events
	for msg := range events {
		switch msg.Status {
		case "start":
			go b.Add(msg.ID)
		case "die":
			go b.RemoveOnExit(msg.ID)
		}
	}

	close(quit)
	log.Fatal("Docker event loop closed") // todo: reconnect?
}
