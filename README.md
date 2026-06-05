# Registrator

Docker service registry bridge.

[![Docker pulls](https://img.shields.io/docker/pulls/fayrus/registrator.svg)](https://hub.docker.com/r/fayrus/registrator/)
[![Docker Image Version](https://img.shields.io/docker/v/fayrus/registrator/latest)](https://hub.docker.com/r/fayrus/registrator/tags)
[![Build and Push Docker Image](https://github.com/fayrus/registrator/actions/workflows/publish.yml/badge.svg)](https://github.com/fayrus/registrator/actions/workflows/publish.yml)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=fayrus_registrator&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=fayrus_registrator)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=fayrus_registrator&metric=coverage)](https://sonarcloud.io/summary/new_code?id=fayrus_registrator)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=fayrus_registrator&metric=bugs)](https://sonarcloud.io/summary/new_code?id=fayrus_registrator)

## Overview

Registrator watches Docker events and keeps a service registry in sync by automatically registering containers when they start and deregistering them when they stop.

Configuration is driven through `SERVICE_` environment variables and labels, so application containers do not need custom registration logic.

## Why this fork exists

This project is a maintained fork of [gliderlabs/registrator](https://github.com/gliderlabs/registrator), a tool that is still useful in many Docker-based environments.

The original project established a simple and effective model for automatic service registration. This fork exists to keep that model practical on current tooling and service-discovery stacks without changing its core behavior unnecessarily.

The goal is to stay close to upstream Registrator in spirit while continuing to maintain the parts that are still valuable in production.

In practice, this fork focuses on:

- keeping the project buildable and releasable on current tooling
- maintaining commonly used registry backends
- documenting supported behavior more clearly
- carrying fixes that are useful in real deployments

## Supported backends

- `consul://`
- `consulkv://`
- `etcd://` for etcd v3 deployments (gRPC, recommended)
- `etcd-legacy://` for legacy etcd v2 setups (HTTP API)
- `zookeeper://`
- `coredns://`

Multi-architecture images are available for `linux/amd64` and `linux/arm64`.

- `etcd://` is the recommended etcd backend for modern deployments.
- The old SkyDNS2 backend was removed in `v8.0.4`. Use `coredns://` instead.
- For `linux/arm/v7`, `linux/arm/v6`, and `linux/386`, use [`v8.0.1`](https://github.com/fayrus/registrator/releases/tag/v8.0.1).

## Quick start

Pull the latest image:

```sh
docker pull fayrus/registrator:latest
```

Run Registrator against Consul:

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:latest \
    consul://localhost:8500
```

Start a sample container:

```sh
docker run -d --name=redis -p 6379:6379 redis
```

Registrator will register `redis` automatically in the configured backend.

For a complete walkthrough, see [Getting Started](https://registrator.fayr.us/getting-started/).

## Documentation

Project documentation is available at **[registrator.fayr.us](https://registrator.fayr.us)**.

Useful starting points:

- [Getting Started](https://registrator.fayr.us/getting-started/)
- [Configuration](https://registrator.fayr.us/configuration/)
- [Consul backend](https://registrator.fayr.us/backends/consul/)
- [etcd backends](https://registrator.fayr.us/backends/etcd/)
- [CoreDNS backend](https://registrator.fayr.us/backends/coredns/)
- [ZooKeeper backend](https://registrator.fayr.us/backends/zookeeper/)

## Scope

This fork stays close to the original Registrator model:

- watch Docker events
- derive service metadata from container labels and environment variables
- publish service registrations to external backends

If your use case needs a full service mesh, sidecar-based discovery, or orchestrator-native service registration, this project is probably not the right abstraction.

If you want a small Docker-native bridge that keeps existing service-discovery workflows working, that is exactly what this project is for.

## License

[MIT](LICENSE)
