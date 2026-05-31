# Getting Started

## Installation

Pull the latest release from [Docker Hub](https://hub.docker.com/r/fayrus/registrator/):

```sh
docker pull fayrus/registrator:latest
```

Version tags are available to pin to a specific release:

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `v9.0.6` | Specific patch version |
| `v9.0` | Latest patch in minor series |
| `v9` | Latest patch in major series |

## Basic usage

Registrator needs access to the Docker socket and must run with `--net=host` to detect the host IP:

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    consul://localhost:8500
```

## Verify it works

Start any container and check your registry:

```sh
docker run -d --name=redis -p 6379:6379 redis
```

Registrator will automatically register a service named `redis` on port `6379`. Check Consul:

```sh
curl http://localhost:8500/v1/catalog/service/redis
```

## Docker Compose example

```yaml
services:
  registrator:
    image: fayrus/registrator:latest
    network_mode: host
    volumes:
      - /var/run/docker.sock:/tmp/docker.sock
    command: consul://localhost:8500
    restart: unless-stopped
```
