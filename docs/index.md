# Registrator

Service registry bridge for Docker.

Registrator watches Docker events and keeps your service registry in sync — automatically registering containers when they start and deregistering them when they stop. No changes to your containers required: configuration is driven entirely by `SERVICE_` environment variables and labels.

Supports [Consul](http://www.consul.io/), [etcd](https://github.com/coreos/etcd), [ZooKeeper](https://zookeeper.apache.org/), and [CoreDNS](https://coredns.io/).

Multi-architecture support: `linux/amd64`, `linux/arm64` — built with Chainguard hardened images.

!!! note
    For `linux/arm/v7`, `linux/arm/v6` and `linux/386` support, use [v8.0.1](https://github.com/fayrus/registrator/releases/tag/v8.0.1).

!!! warning
    The SkyDNS2 backend was removed in v8.0.4 (abandoned since 2016). Use the `coredns://` backend introduced in v9.0.0.

## Quick start

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    consul://localhost:8500
```

See [Getting Started](getting-started.md) for a full walkthrough.
