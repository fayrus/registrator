# Registrator

Registrator watches Docker events and keeps a service registry in sync: it registers containers when they start and deregisters them when they stop. Services are configured through `SERVICE_*` environment variables and labels, so application containers need no registration logic.

Maintained fork of [gliderlabs/registrator](https://github.com/gliderlabs/registrator).

## Quick reference

- **Documentation:** [registrator.fayr.us](https://registrator.fayr.us)
- **Source and issues:** [github.com/fayrus/registrator](https://github.com/fayrus/registrator)
- **Changelog:** [CHANGELOG.md](https://github.com/fayrus/registrator/blob/main/CHANGELOG.md)
- **Architectures:** `linux/amd64`, `linux/arm64`

## Which tag should I use?

Every release publishes four tags that point to the same image:

| Tag | Example | Moves when | Use it for |
| --- | --- | --- | --- |
| `vX.Y.Z` | `v9.2.16` | Never | Production. Reproducible deployments; upgrade deliberately. |
| `vX.Y` | `v9.2` | Each patch release | Receiving fixes automatically within a minor version. |
| `vX` | `v9` | Each minor and patch release | Receiving fixes and new features without breaking changes. |
| `latest` | | Every release, including major versions | Trying Registrator out. Avoid in production. |

All release tags are listed on the [Tags](https://hub.docker.com/r/fayrus/registrator/tags) page, and each release is described in the [GitHub Releases](https://github.com/fayrus/registrator/releases).

**32-bit platforms:** `linux/arm/v7`, `linux/arm/v6`, and `linux/386` are no longer supported. `v8.0.1` is the last image built for them and receives no further fixes. Still running one of these? Comment on [#77](https://github.com/fayrus/registrator/issues/77).

## Which backend should I use?

Pass the registry URI as the last argument. The scheme selects the backend:

| Your registry | URI | Notes |
| --- | --- | --- |
| Consul service catalog | `consul://localhost:8500` | Most common setup |
| Consul KV store | `consulkv://localhost:8500/<path>` | Writes plain key/value entries |
| etcd v3 | `etcd://localhost:2379` | Recommended for etcd; supports TLS |
| etcd v2 | `etcd-legacy://localhost:2379` | Legacy HTTP API only; no TLS |
| CoreDNS (etcd plugin) | `coredns://localhost:2379/skydns?zone=service.local` | Replaces the removed SkyDNS2 backend |
| ZooKeeper | `zookeeper://localhost:2181` | |

`-cleanup` support differs per backend; see the [cleanup support matrix](https://registrator.fayr.us/configuration/#cleanup-support).

## Quick start

Registrator needs the Docker socket and usually host networking, so it can reach a local registry agent:

```sh
docker run -d \
  --name=registrator \
  --net=host \
  --volume=/var/run/docker.sock:/tmp/docker.sock \
  fayrus/registrator:v9 \
    consul://localhost:8500
```

Replace the URI with any backend from the table above. Then start a container with a published port:

```sh
docker run -d --name=redis -p 6379:6379 redis
```

Registrator registers `redis` in the configured backend and removes it when the container stops.

For backend-specific options (etcd TLS, multiple endpoints, CoreDNS zones), see the [backend guides](https://registrator.fayr.us/backends/consul/).

## Common options

Options go before the registry URI:

| Option | Effect |
| --- | --- |
| `-ip=<addr>` | IP to register for ports mapped to the host |
| `-internal` | Register internal container ports instead of published ones |
| `-explicit` | Only register containers that set `SERVICE_NAME` |
| `-cleanup` | Remove dangling services left by previous runs |
| `-resync=<seconds>` | Periodically resynchronize all services |
| `-ttl=<seconds>` / `-ttl-refresh=<seconds>` | Expire services unless refreshed |
| `-retry-attempts=<n>` | Retry the backend connection at startup (`-1` retries forever) |

Per-container settings such as `SERVICE_NAME`, `SERVICE_TAGS`, and health checks are described in the [configuration reference](https://registrator.fayr.us/configuration/).

## Image security

- Runtime image: [Chainguard `static`](https://images.chainguard.dev/directory/image/static/overview), containing only the statically linked `registrator` binary; no shell or package manager.
- Each release is scanned with Trivy (`linux/amd64` build) before publishing; a release with fixable HIGH or CRITICAL vulnerabilities is not published.
- Go code, modules, and the toolchain are checked with `govulncheck` on every pull request and daily.
- Report vulnerabilities privately as described in the [security policy](https://github.com/fayrus/registrator/security/policy).

## License

MIT. See [LICENSE](https://github.com/fayrus/registrator/blob/main/LICENSE).
