# Registrator

Service registry bridge for Docker.

[![Docker pulls](https://img.shields.io/docker/pulls/fayrus/registrator.svg)](https://hub.docker.com/r/fayrus/registrator/)
[![Docker Image Version](https://img.shields.io/docker/v/fayrus/registrator/latest)](https://hub.docker.com/r/fayrus/registrator/tags)
[![Build and Push Docker Image](https://github.com/fayrus/registrator/actions/workflows/docker-publish.yml/badge.svg)](https://github.com/fayrus/registrator/actions/workflows/docker-publish.yml)

Registrator watches Docker events and keeps your service registry in sync — automatically registering containers when they start and deregistering them when they stop. No changes to your containers required: configuration is driven entirely by `SERVICE_` environment variables and labels.

Supports [Consul](http://www.consul.io/), [etcd](https://github.com/coreos/etcd), [ZooKeeper](https://zookeeper.apache.org/), and [CoreDNS](https://coredns.io/).

Multi-architecture support: `linux/amd64`, `linux/arm64` — built with Chainguard hardened images.

> For `linux/arm/v7`, `linux/arm/v6` and `linux/386` support, use [v8.0.1](https://github.com/fayrus/registrator/releases/tag/v8.0.1).

> **Note:** The SkyDNS2 backend was removed in v8.0.4 (abandoned since 2016). It has been replaced by the `coredns://` backend introduced in v9.0.0.

## Getting Started

Pull the latest release from [Docker Hub](https://hub.docker.com/r/fayrus/registrator/):

```sh
docker pull fayrus/registrator:latest
```

Version tags are available to pin to a specific release (e.g. `:v9.0.0`, `:v9.0`, `:v9`).

## Usage

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    consul://localhost:8500
```

## Supported Backends

| URI scheme | Backend | Notes |
|------------|---------|-------|
| `consul://` | HashiCorp Consul | |
| `consul-tls://` | HashiCorp Consul | TLS |
| `consul-unix://` | HashiCorp Consul | Unix socket |
| `etcd://` | etcd | Legacy v2 API |
| `etcd2://` | etcd | v3 client, multi-endpoint, TLS |
| `coredns://` | CoreDNS | Writes SkyDNS-format records to etcd; requires CoreDNS with the `etcd` plugin |
| `zookeeper://` | Apache ZooKeeper | |

### etcd2 — environment variables

| Variable | Description |
|----------|-------------|
| `ETCD_ENDPOINTS` | Comma-separated list of additional etcd endpoints |
| `ETCD_CERT_FILE` | Path to client certificate |
| `ETCD_KEY_FILE` | Path to client key |
| `ETCD_CA_CERT_FILE` | Path to CA certificate |

The `coredns://` backend uses the same TLS variables.

## CLI Options

```
Usage of /bin/registrator:
  /bin/registrator [options] <registry URI>

  -cleanup=false:          Remove dangling services
  -deregister="always":    Deregister exited services "always" or "on-success"
  -explicit=false:         Only register containers that have SERVICE_NAME set
  -internal=false:         Use internal ports instead of published ones
  -ip="":                  IP for ports mapped to the host
  -ip-from-container=false: Use container IP instead of host IP for service registration
  -resync=0:               Frequency with which services are resynchronized
  -useIpFromLabel="":      Use IP stored in the given container label instead of the host IP
  -retry-attempts=0:       Max retry attempts to establish a connection with the backend. Use -1 for infinite retries
  -retry-interval=2000:    Interval (in milliseconds) between retry attempts
  -tags="":                Append tags for all registered services (supports Go template)
  -ttl=0:                  TTL for services (default is no expiry)
  -ttl-refresh=0:          Frequency with which service TTLs are refreshed
```

## Service Configuration

Registrator reads configuration from container environment variables prefixed with `SERVICE_`.

| Variable | Description |
|----------|-------------|
| `SERVICE_NAME` | Override the service name |
| `SERVICE_TAGS` | Comma-separated tags. Supports Go templates (e.g. `{{.Config.Hostname}}`) |
| `SERVICE_<port>_NAME` | Name for a specific port |
| `SERVICE_<port>_<protocol>_NAME` | Name for a specific port and protocol (e.g. `SERVICE_80_tcp_NAME`) |
| `SERVICE_<N>-<M>_IGNORE` | Ignore all ports in range N–M (e.g. `SERVICE_10000-20000_IGNORE=true`) |
| `SERVICE_IGNORE` | Ignore this container entirely |
| `SERVICE_CHECK_HTTP` | HTTP health check path |
| `SERVICE_CHECK_HTTPS` | HTTPS health check path |
| `SERVICE_CHECK_TCP` | TCP health check |
| `SERVICE_CHECK_SCRIPT` | Script health check (space-separated args) |
| `SERVICE_CHECK_GRPC` | gRPC health check |
| `SERVICE_CHECK_TTL` | TTL-based health check |
| `SERVICE_CHECK_INTERVAL` | Health check interval (default `10s`) |
| `SERVICE_CHECK_TIMEOUT` | Health check timeout |
| `SERVICE_CHECK_TLS_SKIP_VERIFY` | Skip TLS verification for health checks |
| `SERVICE_ENABLE_TAG_OVERRIDE` | Allow external agents to update tags in Consul without registrator overwriting them |

## Contributing

Pull requests are welcome. Open a [GitHub issue](https://github.com/fayrus/registrator/issues) to discuss before starting.

## License

MIT
