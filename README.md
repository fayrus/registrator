# Registrator

Service registry bridge for Docker.

[![Docker pulls](https://img.shields.io/docker/pulls/fayrus/registrator.svg)](https://hub.docker.com/r/fayrus/registrator/)
[![Docker Image Version](https://img.shields.io/docker/v/fayrus/registrator/latest)](https://hub.docker.com/r/fayrus/registrator/tags)
[![Build and Push Docker Image](https://github.com/fayrus/registrator/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/fayrus/registrator/actions/workflows/docker-publish.yml)

Registrator automatically registers and deregisters services for any Docker
container by inspecting containers as they come online. Registrator
supports pluggable service registries, which currently includes
[Consul](http://www.consul.io/), [etcd](https://github.com/coreos/etcd),
[ZooKeeper](https://zookeeper.apache.org/) and
[SkyDNS 2](https://github.com/skynetservices/skydns/).

Multi-architecture support: `linux/amd64`, `linux/arm64`, `linux/arm/v7`, `linux/arm/v6`, `linux/386` — built on Go 1.25 and Alpine 3.21.

## Getting Registrator

Get the latest release via [Docker Hub](https://hub.docker.com/r/fayrus/registrator/):

	$ docker pull fayrus/registrator:latest

Latest tag always points to the latest release. There is also a `:master` tag
and version tags to pin to specific releases.

## Using Registrator

Typically, running Registrator looks like this:

    $ docker run -d \
        --name=registrator \
        --net=host \
        --volume=/var/run/docker.sock:/tmp/docker.sock \
        fayrus/registrator:latest \
          consul://localhost:8500

## CLI Options
```
Usage of /bin/registrator:
  /bin/registrator [options] <registry URI>

  -cleanup=false: Remove dangling services
  -deregister="always": Deregister exited services "always" or "on-success"
  -explicit=false: Only register containers which have SERVICE_NAME label set
  -internal=false: Use internal ports instead of published ones
  -ip="": IP for ports mapped to the host
  -resync=0: Frequency with which services are resynchronized
  -useIpFromLabel="": Use IP stored in the given container label instead of the host IP
  -retry-attempts=0: Max retry attempts to establish a connection with the backend. Use -1 for infinite retries
  -retry-interval=2000: Interval (in millisecond) between retry-attempts.
  -tags="": Append tags for all registered services (supports Go template)
  -ttl=0: TTL for services (default is no expiry)
  -ttl-refresh=0: Frequency with which service TTLs are refreshed
```

## Contributing

Pull requests are welcome! Open a [GitHub issue](https://github.com/fayrus/registrator/issues) to discuss before starting.

## License

MIT
