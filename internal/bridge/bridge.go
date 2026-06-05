package bridge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	jsonp "github.com/buger/jsonparser"
	dockerapi "github.com/fsouza/go-dockerclient"
)

var serviceIDPattern = regexp.MustCompile(`^(.+?):([a-zA-Z0-9][a-zA-Z0-9_.-]+):[0-9]+(?::udp)?$`)

const logIgnored = "ignored:"

type Bridge struct {
	sync.Mutex
	registry       RegistryAdapter
	docker         *dockerapi.Client
	services       map[string][]*Service
	deadContainers map[string]*DeadContainer
	config         Config
}

func New(docker *dockerapi.Client, adapterUri string, config Config) (*Bridge, error) {
	uri, err := url.Parse(adapterUri)
	if err != nil {
		return nil, errors.New("bad adapter uri: " + adapterUri)
	}
	factory, found := AdapterFactories.Lookup(uri.Scheme)
	if !found {
		return nil, errors.New("unrecognized adapter: " + adapterUri)
	}

	log.Println("Using", uri.Scheme, "adapter:", uri)
	registry, err := factory.New(uri)
	if err != nil {
		return nil, err
	}
	return &Bridge{
		docker:         docker,
		config:         config,
		registry:       registry,
		services:       make(map[string][]*Service),
		deadContainers: make(map[string]*DeadContainer),
	}, nil
}

func (b *Bridge) Ping() error {
	return b.registry.Ping()
}

func (b *Bridge) Add(containerId string) {
	b.Lock()
	defer b.Unlock()
	b.add(containerId, false)
}

func (b *Bridge) Remove(containerId string) {
	b.remove(containerId, true)
}

func (b *Bridge) RemoveOnExit(containerId string) {
	b.remove(containerId, b.shouldRemove(containerId))
}

func (b *Bridge) Refresh() {
	b.Lock()
	defer b.Unlock()

	for containerId, deadContainer := range b.deadContainers {
		deadContainer.TTL -= b.config.RefreshInterval
		if deadContainer.TTL <= 0 {
			delete(b.deadContainers, containerId)
		}
	}

	for containerId, services := range b.services {
		for _, service := range services {
			err := b.registry.Refresh(service)
			if err != nil {
				log.Println("refresh failed:", service.ID, err)
				continue
			}
			log.Println("refreshed:", containerId[:12], service.ID)
		}
	}
}

func (b *Bridge) Sync(quiet bool) {
	b.Lock()
	defer b.Unlock()

	containers, err := b.docker.ListContainers(dockerapi.ListContainersOptions{})
	if err != nil && quiet {
		log.Println("error listing containers, skipping sync")
		return
	} else if err != nil {
		log.Fatal(err)
	}

	log.Printf("Syncing services on %d containers", len(containers))

	// NOTE: This assumes reregistering will do the right thing, i.e. nothing..
	for _, listing := range containers {
		services := b.services[listing.ID]
		if services == nil {
			b.add(listing.ID, quiet)
		} else {
			b.syncRegisteredServices(services)
		}
	}

	if b.config.Cleanup {
		b.cleanupStaleAndDangling()
	}
}

func (b *Bridge) syncRegisteredServices(services []*Service) {
	for _, service := range services {
		if err := b.registry.Register(service); err != nil {
			log.Println("sync register failed:", service, err)
		}
	}
}

func (b *Bridge) cleanupStaleAndDangling() {
	log.Println("Listing non-exited containers")
	filters := map[string][]string{"status": {"created", "restarting", "running", "paused"}}
	nonExited, err := b.docker.ListContainers(dockerapi.ListContainersOptions{Filters: filters})
	if err != nil {
		log.Println("error listing nonExitedContainers, skipping sync", err)
		return
	}

	b.removeStaleServices(nonExited)

	log.Println("Cleaning up dangling services")
	extServices, err := b.registry.Services()
	if err != nil {
		log.Println("cleanup failed:", err)
		return
	}
	b.removeDanglingServices(extServices)
}

func (b *Bridge) removeStaleServices(nonExited []dockerapi.APIContainers) {
	for listingId := range b.services {
		if !isContainerActive(listingId, nonExited) {
			log.Printf("stale: Removing service %s because it does not exist", listingId)
			go b.RemoveOnExit(listingId)
		}
	}
}

func isContainerActive(id string, containers []dockerapi.APIContainers) bool {
	for _, c := range containers {
		if id == c.ID {
			return true
		}
	}
	return false
}

func (b *Bridge) removeDanglingServices(extServices []*Service) {
Outer:
	for _, extService := range extServices {
		matches := serviceIDPattern.FindStringSubmatch(extService.ID)
		if len(matches) != 3 {
			continue
		}
		if matches[1] != Hostname {
			continue
		}
		serviceContainerName := matches[2]
		for _, listing := range b.services {
			for _, service := range listing {
				if service.Name == extService.Name && serviceContainerName == service.Origin.container.Name[1:] {
					continue Outer
				}
			}
		}
		log.Println("dangling:", extService.ID)
		if err := b.registry.Deregister(extService); err != nil {
			log.Println("deregister failed:", extService.ID, err)
			continue
		}
		log.Println(extService.ID, "removed")
	}
}

func (b *Bridge) add(containerId string, quiet bool) {
	if d := b.deadContainers[containerId]; d != nil {
		b.services[containerId] = d.Services
		delete(b.deadContainers, containerId)
	}

	if b.services[containerId] != nil {
		log.Println("container, ", containerId[:12], ", already exists, ignoring")
		// Alternatively, remove and readd or resubmit.
		return
	}

	container, err := b.docker.InspectContainerWithOptions(dockerapi.InspectContainerOptions{ID: containerId})
	if err != nil {
		log.Println("unable to inspect container:", containerId[:12], err)
		return
	}

	ports := extractContainerPorts(container)

	if len(ports) == 0 && !quiet {
		log.Println(logIgnored, container.ID[:12], "no published ports")
		return
	}

	servicePorts := b.filterPublishedPorts(ports, quiet, container.ID)

	isGroup := len(servicePorts) > 1
	for _, port := range servicePorts {
		service, err := b.newService(port, isGroup)
		if err != nil {
			log.Println(logIgnored, container.ID[:12], "service on port", port.ExposedPort, err)
			continue
		}
		if service == nil {
			if !quiet {
				log.Println(logIgnored, container.ID[:12], "service on port", port.ExposedPort)
			}
			continue
		}
		if err = b.registry.Register(service); err != nil {
			log.Println("register failed:", service, err)
			continue
		}
		b.services[container.ID] = append(b.services[container.ID], service)
		log.Println("added:", container.ID[:12], service.ID)
	}
}

func extractContainerPorts(container *dockerapi.Container) map[string]ServicePort {
	ports := make(map[string]ServicePort)
	for port := range container.Config.ExposedPorts {
		published := []dockerapi.PortBinding{{HostIP: "0.0.0.0", HostPort: port.Port()}}
		ports[string(port)] = servicePort(container, port, published)
	}
	for port, published := range container.NetworkSettings.Ports {
		ports[string(port)] = servicePort(container, port, published)
	}
	return ports
}

func (b *Bridge) filterPublishedPorts(ports map[string]ServicePort, quiet bool, containerID string) map[string]ServicePort {
	servicePorts := make(map[string]ServicePort)
	for key, port := range ports {
		if !b.config.Internal && port.HostPort == "" {
			if !quiet {
				log.Println(logIgnored, containerID[:12], "port", port.ExposedPort, "not published on host")
			}
			continue
		}
		servicePorts[key] = port
	}
	return servicePorts
}

func (b *Bridge) newService(port ServicePort, isgroup bool) (*Service, error) {
	container := port.container
	defaultName := strings.Split(path.Base(container.Config.Image), ":")[0]

	port, hostname := resolvePortHostIP(port)
	if b.config.HostIp != "" {
		port.HostIP = b.config.HostIp
	}

	metadata, metadataFromPort := serviceMetaData(container.Config, port.ExposedPort, port.PortType)

	if mapDefault(metadata, "ignore", "") != "" {
		return nil, nil
	}

	serviceName := mapDefault(metadata, "name", "")
	if serviceName == "" {
		if b.config.Explicit {
			return nil, nil
		}
		serviceName = defaultName
	}

	service := new(Service)
	service.Origin = port
	service.ID = hostname + ":" + container.Name[1:] + ":" + port.ExposedPort
	service.Name = serviceName
	if isgroup && !metadataFromPort["name"] {
		service.Name += "-" + port.ExposedPort
	}

	if b.config.Internal {
		service.IP = port.ExposedIP
		service.Port, _ = strconv.Atoi(port.ExposedPort)
	} else {
		service.IP = port.HostIP
		service.Port, _ = strconv.Atoi(port.HostPort)
	}

	b.resolveServiceIP(service, port, container)

	ForceTags := b.config.ForceTags
	if len(ForceTags) != 0 {
		var err error
		ForceTags, err = executeTagTemplate(ForceTags, container)
		if err != nil {
			return nil, fmt.Errorf("force tags template failed: %w", err)
		}
	}

	serviceTags := mapDefault(metadata, "tags", "")
	if len(serviceTags) != 0 {
		var err error
		serviceTags, err = executeTagTemplate(serviceTags, container)
		if err != nil {
			return nil, fmt.Errorf("service tags template failed: %w", err)
		}
		metadata["tags"] = serviceTags
	}

	if port.PortType == "udp" {
		service.Tags = combineTags(mapDefault(metadata, "tags", ""), ForceTags, "udp")
		service.ID = service.ID + ":udp"
	} else {
		service.Tags = combineTags(mapDefault(metadata, "tags", ""), ForceTags)
	}

	if id := mapDefault(metadata, "id", ""); id != "" {
		service.ID = id
	}

	delete(metadata, "id")
	delete(metadata, "tags")
	delete(metadata, "name")
	service.Attrs = metadata
	service.TTL = b.config.RefreshTtl

	return service, nil
}

// resolvePortHostIP resolves 0.0.0.0 host bindings to the actual host IP
// and returns the updated port along with the hostname used for service IDs.
func resolvePortHostIP(port ServicePort) (ServicePort, string) {
	hostname := Hostname
	if hostname == "" {
		hostname = port.HostIP
	}
	if port.HostIP == "0.0.0.0" {
		if ip, err := net.ResolveIPAddr("ip", hostname); err == nil {
			port.HostIP = ip.String()
		}
	}
	return port, hostname
}

// resolveServiceIP applies IP override options in priority order:
// IpFromContainer → UseIpFromLabel → container NetworkMode.
func (b *Bridge) resolveServiceIP(service *Service, port ServicePort, container *dockerapi.Container) {
	if b.config.IpFromContainer {
		service.IP = port.ExposedIP
	}

	if b.config.UseIpFromLabel != "" {
		containerIp := container.Config.Labels[b.config.UseIpFromLabel]
		if containerIp != "" {
			if slashIndex := strings.LastIndex(containerIp, "/"); slashIndex > -1 {
				service.IP = containerIp[:slashIndex]
			} else {
				service.IP = containerIp
			}
			log.Println("using container IP " + service.IP + " from label '" + b.config.UseIpFromLabel + "'")
		} else {
			log.Println("Label '" + b.config.UseIpFromLabel + "' not found in container configuration")
		}
	}

	// NetworkMode can point to another container (kubernetes pods)
	if strings.HasPrefix(container.HostConfig.NetworkMode, "container:") {
		networkContainerId := strings.Split(container.HostConfig.NetworkMode, ":")[1]
		log.Println(service.Name + ": detected container NetworkMode, linked to: " + networkContainerId[:12])
		networkContainer, err := b.docker.InspectContainerWithOptions(dockerapi.InspectContainerOptions{ID: networkContainerId})
		if err != nil {
			log.Println("unable to inspect network container:", networkContainerId[:12], err)
			return
		}
		service.IP = networkContainer.NetworkSettings.IPAddress
		log.Println(service.Name + ": using network container IP " + service.IP)
	}
}

func executeTagTemplate(tmplStr string, container *dockerapi.Container) (string, error) {
	tmpl, err := template.New("tags").Funcs(buildTagFuncMap()).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, container); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func buildTagFuncMap() template.FuncMap {
	return template.FuncMap{
		// strSlice slices a string from start to end (same as s[start:end]).
		"strSlice": func(v string, i ...int) string {
			if len(i) == 1 {
				if len(v) >= i[0] {
					return v[i[0]:]
				}
				return v
			}
			if len(i) == 2 && len(v) >= i[0] && len(v) >= i[1] {
				if i[0] == 0 {
					return v[:i[1]]
				}
				if i[1] < i[0] {
					return v[i[0]:]
				}
				return v[i[0]:i[1]]
			}
			return v
		},
		// sIndex returns element i from slice s; negative i counts from the end.
		"sIndex": func(i int, s []string) string {
			if i < 0 {
				i = i * -1
				if i >= len(s) {
					return s[0]
				}
				return s[len(s)-i]
			}
			if i >= len(s) {
				return s[len(s)-1]
			}
			return s[i]
		},
		// mIndex returns the value for key k in map m.
		"mIndex": func(k string, m map[string]string) string {
			return m[k]
		},
		"toUpper": func(v string) string { return strings.ToUpper(v) },
		"toLower": func(v string) string { return strings.ToLower(v) },
		// replace replaces n occurrences of old with new in v.
		"replace": func(n int, old, new, v string) string {
			return strings.Replace(v, old, new, n)
		},
		// join joins slice s with sep.
		"join": func(sep string, s []string) string { return strings.Join(s, sep) },
		// split splits v by sep.
		"split": func(sep, v string) []string { return strings.Split(v, sep) },
		// splitIndex splits v by sep and returns element i.
		"splitIndex": func(i int, sep, v string) string {
			l := strings.Split(v, sep)
			if i < 0 {
				i = i * -1
				if i >= len(l) {
					return l[0]
				}
				return l[len(l)-i]
			}
			if i >= len(l) {
				return l[len(l)-1]
			}
			return l[i]
		},
		// matchFirstElement returns the first element of s matching regex r.
		"matchFirstElement": func(r string, s []string) string {
			re := regexp.MustCompile(r)
			for _, e := range s {
				if re.MatchString(e) {
					return e
				}
			}
			return ""
		},
		// matchAllElements returns all elements of s matching regex r.
		"matchAllElements": func(r string, s []string) []string {
			var m []string
			re := regexp.MustCompile(r)
			for _, e := range s {
				if re.MatchString(e) {
					m = append(m, e)
				}
			}
			return m
		},
		// httpGet fetches a URL and returns the body bytes.
		"httpGet": func(url string) []byte {
			c := &http.Client{Timeout: 10 * time.Second}
			res, err := c.Get(url)
			if err != nil {
				log.Printf("httpGet template function encountered an error while executing HTTP request: %v", err)
				return []byte("")
			}
			body, err := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if err != nil {
				log.Printf("httpGet template function encountered an error while reading HTTP body payload: %v", err)
				return []byte("")
			}
			return body
		},
		// jsonParse extracts a value from b using double-colon-separated keys.
		"jsonParse": func(b []byte, k string) string {
			keys := strings.Split(k, "::")
			js, _, _, err := jsonp.Get(b, keys...)
			if err != nil {
				log.Printf("jsonParse template function encountered an error while parsing JSON object %v: %v", keys, err)
			}
			return string(js)
		},
	}
}

func (b *Bridge) remove(containerId string, deregister bool) {
	b.Lock()
	defer b.Unlock()

	if deregister {
		deregisterAll := func(services []*Service) {
			for _, service := range services {
				err := b.registry.Deregister(service)
				if err != nil {
					log.Println("deregister failed:", service.ID, err)
					continue
				}
				log.Println("removed:", containerId[:12], service.ID)
			}
		}
		deregisterAll(b.services[containerId])
		if d := b.deadContainers[containerId]; d != nil {
			deregisterAll(d.Services)
			delete(b.deadContainers, containerId)
		}
	} else if b.config.RefreshTtl != 0 && b.services[containerId] != nil {
		// need to stop the refreshing, but can't delete it yet
		b.deadContainers[containerId] = &DeadContainer{b.config.RefreshTtl, b.services[containerId]}
	}
	delete(b.services, containerId)
}

// bit set on ExitCode if it represents an exit via a signal
var dockerSignaledBit = 128

func (b *Bridge) shouldRemove(containerId string) bool {
	if b.config.DeregisterCheck == "always" {
		return true
	}
	container, err := b.docker.InspectContainerWithOptions(dockerapi.InspectContainerOptions{ID: containerId})
	if _, ok := err.(*dockerapi.NoSuchContainer); ok {
		// the container has already been removed from Docker
		// e.g. probabably run with "--rm" to remove immediately
		// so its exit code is not accessible
		log.Printf("registrator: container %v was removed, could not fetch exit code", containerId[:12])
		return true
	}

	switch {
	case err != nil:
		log.Printf("registrator: error fetching status for container %v on \"die\" event: %v\n", containerId[:12], err)
		return false
	case container.State.Running:
		log.Printf("registrator: not removing container %v, still running", containerId[:12])
		return false
	case container.State.ExitCode == 0:
		return true
	case container.State.ExitCode&dockerSignaledBit == dockerSignaledBit:
		return true
	}
	return false
}

var Hostname string

func init() {
	// It's ok for Hostname to ultimately be an empty string
	// An empty string will fall back to trying to make a best guess
	Hostname, _ = os.Hostname()
}
