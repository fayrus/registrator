# Change Log

All notable changes to this project will be documented in this file.

## [v9.2.0](https://github.com/fayrus/registrator/releases/tag/v9.2.0) - 2026-06-26

### Added
- Added `-cleanup` listing support for `consulkv://`, modern `etcd://`, legacy `etcd-legacy://`, and `zookeeper://`
- Added shared KV service parsing for ConsulKV and etcd-style backends
- Added tests for cleanup service listing, malformed registry entries, and backend listing failures

### Changed
- Documented backend-specific `-cleanup` behavior and limitations across Consul, ConsulKV, etcd, etcd-legacy, ZooKeeper, and CoreDNS
- ZooKeeper registrations now preserve the Registrator service ID in the znode payload so new registrations can be cleaned safely

### Notes
- `coredns://` remains unsupported for `-cleanup` because SkyDNS keys do not preserve the original Registrator service ID safely
- ZooKeeper cleanup applies to registrations whose payload includes the service ID; older registrations without that field are ignored

Closes #47.

## [v9.1.1](https://github.com/fayrus/registrator/releases/tag/v9.1.1) - 2026-06-05

### Changed
- Reduced cognitive complexity across all SonarCloud-flagged methods to ≤ 15:
  - `internal/bridge/bridge.go`: `Sync`, `add`, `newService`, `executeTagTemplate`, `removeDanglingServices`
  - `internal/bridge/util.go`: `serviceMetaData`
  - `backends/consul/consul.go`: `buildCheck`
  - `registrator.go`: `main`
- Defined `logIgnored` constant to replace duplicated `"ignored:"` literal
- Quoted `$TARGETOS`, `$TARGETARCH`, `$TARGETVARIANT` variables in Dockerfile `RUN` instruction

## [v9.1.0](https://github.com/fayrus/registrator/releases/tag/v9.1.0) - 2026-06-04

### Changed
- etcd backend URI schemes renamed to match the actual API used:
  - `etcd://` now maps to the modern gRPC v3 backend (previously `etcd2://`)
  - `etcd-legacy://` now maps to the legacy HTTP v2 backend (previously `etcd://`)

### Migration

| Old URI | New URI | Notes |
|---------|---------|-------|
| `etcd2://host:2379` | `etcd://host:2379` | Modern backend, gRPC v3 — most users |
| `etcd://host:2379` | `etcd-legacy://host:2379` | Legacy backend, HTTP v2 — requires `--enable-v2=true` |

Closes #43.

## [v9.0.11](https://github.com/fayrus/registrator/releases/tag/v9.0.11) - 2026-06-03

### Fixed
- ConsulKV: `consulkv-unix` URI parsing now validates the path format before indexing, returning a descriptive error instead of panicking on malformed URIs
- ConsulKV: `Register` and `Deregister` no longer panic when the adapter path is empty or is the root `/`

### Changed
- All existing `golangci-lint` findings resolved: replaced deprecated `ioutil` with `io`, switched to `strings.ReplaceAll`, removed deprecated `grpc.WithBlock()`, fixed unchecked error returns, and removed unused `retry` and `getopt` functions
- Lint workflow now runs without `only-new-issues`, enforcing the full ruleset on every PR
- Removed unused `github.com/cenkalti/backoff` dependency; `google.golang.org/grpc` demoted to indirect
- Added `make tidy` target

## [v9.0.10](https://github.com/fayrus/registrator/releases/tag/v9.0.10) - 2026-06-03

### Fixed
- ConsulKV: replace unsafe `path[1:]` slice with `strings.TrimPrefix` in `Register` and `Deregister` to avoid panic or incorrect path when the configured path lacks a leading slash
- ConsulKV: remove leftover debug log statements from `Register`
- ZooKeeper: `Factory.New` now returns an error early if `Exists` fails, instead of continuing with an incorrect `exists=false` value
- ZooKeeper: `Factory.New` now propagates the error from `Create` instead of discarding it silently
- ZooKeeper: service registration nodes are now created as persistent (`flags=0`) instead of ephemeral (`flags=1`), preventing unintended node deletion on session loss
- Consul: rename local variable `deregister_after` to `deregisterAfter` to follow Go naming conventions

### Changed
- CI pipelines restructured: unified test and lint workflow, improved publish pipeline with dedicated security scan stage, and path filters added to avoid unnecessary runs on documentation-only changes

## [v9.0.9](https://github.com/fayrus/registrator/releases/tag/v9.0.9) - 2026-05-31

### Fixed
- Consul `check_script` with whitespace-only value no longer registers an empty args check — it is now treated as no check
- `google/shlex` promoted to direct dependency in `go.mod`
- Added regression tests for quoted arguments, malformed scripts, and empty input in `check_script`

## [v9.0.8](https://github.com/fayrus/registrator/releases/tag/v9.0.8) - 2026-05-31

### Fixed
- Consul `check_script` is now parsed with `google/shlex` instead of `strings.Split`, correctly handling quoted arguments and embedded spaces. Malformed scripts are logged and the check is skipped rather than sending incorrect tokens to Consul.

### Changed
- Updated `FTP-Deploy-Action` from v4.3.4 to v4.4.0 to address Node.js 20 deprecation in the docs deploy workflow

## [v9.0.7](https://github.com/fayrus/registrator/releases/tag/v9.0.7) - 2026-05-31

### Changed
- Added `golangci-lint` to the pull request workflow with `only-new-issues` mode — enforces `errcheck`, `govet`, `staticcheck`, and `unused` on new code
- Added `make lint` target for local use
- Updated `actions/setup-python` from v5 to v6 in the docs deploy workflow to address Node.js 20 deprecation
- Fixed `requirements-docs.txt` — removed inline hash that was incorrectly activating `--require-hashes` mode in pip

## [v9.0.6](https://github.com/fayrus/registrator/releases/tag/v9.0.6) - 2026-05-31

### Changed
- Extracted `zkClient`, `kvStore`, and `etcd2Client` interfaces in ZooKeeper, ConsulKV, and etcd2 backends to decouple adapters from concrete client implementations
- Added unit tests for ZooKeeper, ConsulKV, and etcd2 backends covering `Register`, `Deregister`, `Ping`, and error propagation
- `requirements-docs.txt` excluded from build and test workflow triggers to avoid spurious CI runs

## [v9.0.5](https://github.com/fayrus/registrator/releases/tag/v9.0.5) - 2026-05-31

### Fixed
- ZooKeeper `Register` now returns errors from each step (`Exists`, base path `Create`, `json.Marshal`, service path `Create`) instead of silently succeeding on partial failures
- ZooKeeper `Deregister` returns immediately on delete failure, preventing the subsequent `Children` call from overwriting the error and masking stale state
- ZooKeeper znode payload now stores the correct `ContainerID` instead of `ContainerHostname`

## [v9.0.4](https://github.com/fayrus/registrator/releases/tag/v9.0.4) - 2026-05-31

### Security
- Upgraded `golang.org/x/net` v0.53.0 → v0.55.0 to address 6 CVEs including one CRITICAL (CVE-2026-39821, score 9.6)
- Upgraded `golang.org/x/sys` v0.43.0 → v0.45.0 to address CVE-2026-39824 (score 3.3)
- `golang.org/x/text` updated transitively v0.36.0 → v0.37.0

## [v9.0.3](https://github.com/fayrus/registrator/releases/tag/v9.0.3) - 2026-05-24

### Fixed
- Invalid tag templates in `SERVICE_TAGS` or the global `-tags` flag no longer terminate the process. Registrator now logs the template error and skips only the affected service registration.

### Changed
- Tag template evaluation now returns errors to the bridge layer instead of calling `log.Fatal`, so malformed container metadata is handled as a per-service failure.

## [v9.0.2](https://github.com/fayrus/registrator/releases/tag/v9.0.2) - 2026-05-17

### Fixed
- Backend constructors no longer call `log.Fatal` or `panic` on initialization errors — errors are now propagated and retried via `-retry-attempts` and `-retry-interval`

### Changed
- `AdapterFactory.New` now returns `(RegistryAdapter, error)` to allow callers to handle initialization failures gracefully
- Introduced `connectWithRetry` in main, wrapping both backend construction and `Ping` in the retry loop

## [v9.0.1](https://github.com/fayrus/registrator/releases/tag/v9.0.1) - 2026-05-10

### Security
- Pinned builder to `cgr.dev/chainguard/go:1.26.3` to address CVEs in `golang/stdlib` 1.26.2

### Changed
- Updated `go.mod` to Go 1.26.3 to align CI and Docker build toolchain
- CI restructured: tests run on pull requests, build and push on merge to `main`

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

[unreleased]: https://github.com/fayrus/registrator/compare/v9.2.0...HEAD
[v9.2.0]: https://github.com/fayrus/registrator/compare/v9.1.1...v9.2.0
[v9.1.1]: https://github.com/fayrus/registrator/compare/v9.1.0...v9.1.1
[v9.1.0]: https://github.com/fayrus/registrator/compare/v9.0.11...v9.1.0
[v9.0.11]: https://github.com/fayrus/registrator/compare/v9.0.10...v9.0.11
[v9.0.10]: https://github.com/fayrus/registrator/compare/v9.0.9...v9.0.10
[v9.0.9]: https://github.com/fayrus/registrator/compare/v9.0.8...v9.0.9
[v9.0.8]: https://github.com/fayrus/registrator/compare/v9.0.7...v9.0.8
[v9.0.7]: https://github.com/fayrus/registrator/compare/v9.0.6...v9.0.7
[v9.0.6]: https://github.com/fayrus/registrator/compare/v9.0.5...v9.0.6
[v9.0.5]: https://github.com/fayrus/registrator/compare/v9.0.4...v9.0.5
[v9.0.4]: https://github.com/fayrus/registrator/compare/v9.0.3...v9.0.4
[v9.0.3]: https://github.com/fayrus/registrator/compare/v9.0.2...v9.0.3
[v9.0.2]: https://github.com/fayrus/registrator/compare/v9.0.1...v9.0.2
[v9.0.1]: https://github.com/fayrus/registrator/compare/v9.0.0...v9.0.1
[v9.0.0]: https://github.com/fayrus/registrator/compare/v8.0.4...v9.0.0
[v8.0.4]: https://github.com/fayrus/registrator/compare/v8.0.3...v8.0.4
[v8.0.3]: https://github.com/fayrus/registrator/compare/v8.0.2...v8.0.3
[v8.0.2]: https://github.com/fayrus/registrator/compare/v8.0.1...v8.0.2
[v8.0.1]: https://github.com/fayrus/registrator/compare/v8.0.0...v8.0.1
[v8.0.0]: https://github.com/fayrus/registrator/releases/tag/v8.0.0
