# Change Log

All notable changes to this project will be documented in this file.

## [v9.0.0](https://github.com/fayrus/registrator/releases/tag/v9.0.0) - 2026-04-22

### Added
- `SERVICE_ENABLE_TAG_OVERRIDE=true` label support for Consul — allows external agents to update service tags without registrator overwriting them
- `-ip-from-container` flag — uses the container's internal IP instead of the host IP for service registration, avoiding duplicate registrations with Traefik and similar systems (inspired by @colcek in gliderlabs/registrator#703)
- `SERVICE_<port>_<protocol>_<key>` encoding support — allows differentiating metadata between the same port on different protocols (e.g. `SERVICE_8080_tcp_NAME=web`) (inspired by @pmundt in gliderlabs/registrator#668)
- Port range support in `SERVICE_` variables — `SERVICE_10000-20000_IGNORE=true` applies to all ports in the range, useful for containers exposing many UDP ports (e.g. VoIP/RTP) (inspired by @afr1983 in gliderlabs/registrator#383)
- Go template support in `SERVICE_TAGS` per container — e.g. `SERVICE_TAGS=host-{{.Config.Hostname}},web` (same template functions as the global `-tags` flag) (inspired by @alinoeabrassart in gliderlabs/registrator#503 and @psyhomb in gliderlabs/registrator#677)
- New `etcd2://` backend using the official etcd v3 client (`go.etcd.io/etcd/client/v3`) — supports multiple endpoints via `ETCD_ENDPOINTS` env var and TLS via `ETCD_CERT_FILE`, `ETCD_KEY_FILE`, `ETCD_CA_CERT_FILE`. The existing `etcd://` backend is unchanged (inspired by @woshihaoren in gliderlabs/registrator#649)
- New `coredns://` backend — writes service records to etcd in SkyDNS format so CoreDNS can resolve them via its `etcd` plugin. Usage: `coredns://etcd-host:2379/skydns?zone=service.local`

### Changed
- Restructured project layout: bridge package moved to `internal/bridge/`, registry backends moved to `backends/` (`consul`, `consulkv`, `etcd`, `zookeeper`)
- Updated all dependencies to latest versions

---

## [v8.0.4](https://github.com/fayrus/registrator/releases/tag/v8.0.4) - 2026-04-21

### Fixed
- `SERVICE_CHECK_SCRIPT` was being passed to Consul as a single argument instead of splitting by spaces, causing the check to always fail

### Changed
- `check_timeout` and `check_tls_skip_verify` are now applied to all check types (HTTP, HTTPS, TCP, gRPC, script). Previously `check_tls_skip_verify` only worked with gRPC checks and `check_timeout` was duplicated per check type

### Removed
- Dropped SkyDNS2 backend — the project has been abandoned since 2016 and its Docker image is incompatible with modern container runtimes. [CoreDNS](https://coredns.io/) is its successor and will be evaluated as a replacement in a future release

## [v8.0.3](https://github.com/fayrus/registrator/releases/tag/v8.0.3) - 2026-04-20

### Changed
- Reduced supported platforms to `linux/amd64` and `linux/arm64` (Chainguard images do not support `linux/arm/v7`, `linux/arm/v6` or `linux/386`)

## [v8.0.2](https://github.com/fayrus/registrator/releases/tag/v8.0.2) - 2026-04-20

### Changed
- Switched to Chainguard hardened base images (`cgr.dev/chainguard/go` and `cgr.dev/chainguard/static`)

## [v8.0.1](https://github.com/fayrus/registrator/releases/tag/v8.0.1) - 2026-04-20

### Changed
- Updated `buger/jsonparser` to v1.1.2
- Last version with support for `linux/arm/v7`, `linux/arm/v6` and `linux/386`

## [v8.0.0](https://github.com/fayrus/registrator/releases/tag/v8.0.0) - 2026-04-19

### Added
- Multi-architecture Docker image support: `linux/amd64`, `linux/arm64`, `linux/arm/v7`, `linux/arm/v6`, `linux/386`
- GitHub Actions workflow for automated multi-arch build and push to Docker Hub

### Removed
- CircleCI pipeline (`config.yml`)
- Development Dockerfile (`Dockerfile.dev`)

### Changed
- Project moved to independent maintenance under `fayrus/registrator`
- Upgraded builder image: `golang:1.17.1-alpine3.14` => `golang:1.25-alpine`
- Upgraded runtime image: `alpine:3.14` => `alpine:3.21`
- Upgraded Go module path to `github.com/fayrus/registrator`
- Updated all dependencies to latest compatible versions
- Makefile updated: `release` target now uses `gh` CLI


---

For history prior to v8.0.0, see the upstream projects:
- [psyhomb/registrator](https://github.com/psyhomb/registrator)
- [gliderlabs/registrator](https://github.com/gliderlabs/registrator)

[unreleased]: https://github.com/fayrus/registrator/compare/v9.0.0...HEAD
[v9.0.0]: https://github.com/fayrus/registrator/compare/v8.0.4...v9.0.0
[v8.0.4]: https://github.com/fayrus/registrator/compare/v8.0.3...v8.0.4
[v8.0.3]: https://github.com/fayrus/registrator/compare/v8.0.2...v8.0.3
[v8.0.2]: https://github.com/fayrus/registrator/compare/v8.0.1...v8.0.2
[v8.0.1]: https://github.com/fayrus/registrator/compare/v8.0.0...v8.0.1
[v8.0.0]: https://github.com/fayrus/registrator/releases/tag/v8.0.0
