# Change Log

All notable changes to this project will be documented in this file.

## [Unreleased]

## [v9.2.18](https://github.com/fayrus/registrator/releases/tag/v9.2.18) - 2026-11-24

### Fixed
- A restarted container (restart policy or `docker restart`) is no longer deregistered while it is running; this also applies with the default `-deregister=always` ([#99](https://github.com/fayrus/registrator/issues/99))

### Added
- Tests for the container registration lifecycle (add, deregistration on exit, dead-container TTL, refresh, sync, stale and dangling cleanup); `internal/bridge/bridge.go` now counts toward the SonarCloud coverage gate ([#95](https://github.com/fayrus/registrator/issues/95))

### Changed
- The published image is exactly the one Trivy scanned: the release is built once with the runtime base pinned to a digest, both `linux/amd64` and `linux/arm64` are scanned, and only then are the release tags applied; the base digest is recorded in the `org.opencontainers.image.base.digest` label ([#93](https://github.com/fayrus/registrator/issues/93))
- Changes that only touch Go test files (`*_test.go`) no longer start an image publish ([#97](https://github.com/fayrus/registrator/issues/97))

## [v9.2.17](https://github.com/fayrus/registrator/releases/tag/v9.2.17) - 2026-11-24

### Changed
- Updated the etcd client modules (`go.etcd.io/etcd/api/v3`, `client/v3`, `client/pkg/v3`) to v3.7.4; the release contains etcd server fixes only, with no client behavior change
- `linux/arm/v7`, `linux/arm/v6`, and `linux/386` are documented as no longer supported (`v8.0.1` was the last image for them); remaining users are asked to comment on [#77](https://github.com/fayrus/registrator/issues/77)
- The Docker Hub page now shows an operator-focused description (tag selection, backend choice, quick start, image security) instead of the repository README ([#76](https://github.com/fayrus/registrator/issues/76))
- Publishing fails before building when the `VERSION` tag is already on Docker Hub, so a released image tag is never overwritten; publish runs no longer overlap ([#81](https://github.com/fayrus/registrator/issues/81))
- The publish workflow scans images with Trivy `v0.76.2` (was `v0.74.0`)
- CI checks (`Lint (Actions)`, `Tests (Go)`, `Lint (Go)`) report on every pull request and are skipped when only documentation changes; tests run with the race detector (also in `make test` and `mise run test`), and golangci-lint is pinned to `v2.15.0` ([#83](https://github.com/fayrus/registrator/issues/83))
- Dependabot checks all ecosystems weekly on Monday, groups only minor and patch updates, uses conventional commit prefixes, and no longer proposes minor or major `golang` builder image upgrades ([#85](https://github.com/fayrus/registrator/issues/85))

### Added
- `mise.toml` pins the development toolchain (Go, golangci-lint, govulncheck, Trivy, actionlint, shellcheck) and defines `test`, `coverage`, `lint`, `vulncheck`, `scan`, and `actionlint` tasks; the `Makefile` keeps working ([#74](https://github.com/fayrus/registrator/issues/74))
- CI runs `govulncheck` on every pull request and daily on `main`, using the version pinned in `mise.toml` ([#75](https://github.com/fayrus/registrator/issues/75))
- Security policy with private vulnerability reporting, contributing guide, code of conduct, and pull request template ([#87](https://github.com/fayrus/registrator/issues/87))

## [v9.2.16](https://github.com/fayrus/registrator/releases/tag/v9.2.16) - 2026-11-16

### Fixed
- Image labels now identify the project: `org.opencontainers.image.version` carries the release version instead of `latest`, `authors` and `vendor` are set to Fayrus instead of being inherited from the Chainguard base, and the Chainguard-specific `dev.chainguard.*` labels are blanked
- Added `org.opencontainers.image.base.name` pointing at `cgr.dev/chainguard/static:latest`

### Changed
- The builder stage uses the pinned `golang:1.27.4-alpine` image instead of the floating `cgr.dev/chainguard/go:latest`, so builds use the Go version declared in `go.mod`; the runtime image is unchanged
- Enabled Dependabot version updates for the Dockerfile base images

## [v9.2.15](https://github.com/fayrus/registrator/releases/tag/v9.2.15) - 2026-11-15

### Changed
- Updated Go dependencies: `go.etcd.io/etcd/*` v3.7.1 → v3.7.3, `github.com/hashicorp/consul/api` v1.34.4 → v1.35.0, `github.com/fsouza/go-dockerclient` v1.13.2 → v1.13.3 and `github.com/stretchr/testify` v1.11.1 → v1.12.1 (PR #71)
- Enabled Dependabot version updates for Go modules, GitHub Actions and the docs requirements (PR #69)
- Updated GitHub Actions (`actions/setup-python` v7.1.0, `github/codeql-action` v4.39.3, `docker/login-action` v4.8.0, `SonarSource/sonarqube-scan-action` v8.5.0) and `mkdocs-material` 9.8.1 for the docs site (PRs #70, #72)

## [v9.2.14](https://github.com/fayrus/registrator/releases/tag/v9.2.14) - 2026-11-15

### Fixed
- `etcd://` and `coredns://` now enable TLS when only `ETCD_CA_CERT_FILE` is set (server-authenticated TLS without a client certificate); previously the CA file was ignored and the client silently connected in plain text (R-050)

### Changed
- Setting only one of `ETCD_CERT_FILE` / `ETCD_KEY_FILE` is now a startup error instead of silently disabling TLS
- The etcd client logs at startup whether TLS is enabled

Closes #67

## [v9.2.13](https://github.com/fayrus/registrator/releases/tag/v9.2.13) - 2026-11-15

### Fixed
- Containers using a `container:<name>` network mode with a reference shorter than 12 characters (e.g. `--network container:db`) no longer crash the process with a slice-bounds panic; container IDs in log messages are now truncated safely (R-046)

Closes #65

## [v9.2.12](https://github.com/fayrus/registrator/releases/tag/v9.2.12) - 2026-11-15

### Security
- Bumped Go `1.27.3` → `1.27.4` to address CVE-2026-97054 (GO-2026-6859) in `net/http`

## [v9.2.11](https://github.com/fayrus/registrator/releases/tag/v9.2.11) - 2026-11-10

### Security
- Upgraded `github.com/klauspost/compress` v1.19.1 → v1.20.4 to address CVE-2026-116591 (GO-2026-6865), a panic when decompressing malformed zstd frames reachable through the Docker client
- Upgraded `certifi` 2026.5.20 → 2026.10.23 and `urllib3` 2.7.0 → 2.8.0 in the documentation toolchain (Dependabot alerts, PR #62); the binary is not affected

### Changed
- The SonarCloud workflow now skips its analysis job instead of the whole workflow on docs-only pull requests, so the required `sonarcloud` check no longer blocks them (PR #62)

## [v9.2.10](https://github.com/fayrus/registrator/releases/tag/v9.2.10) - 2026-10-29

### Security
- Upgraded `golang.org/x/net` v0.61.0 → v0.62.0 to address CVE-2026-78661 (GO-2026-6860) and CVE-2026-97045 (GO-2026-6866) in `x/net/html`; registrator does not call the affected code

## [v9.2.9](https://github.com/fayrus/registrator/releases/tag/v9.2.9) - 2026-10-25

### Security
- Bumped Go `1.27.2` → `1.27.3` to address CVE-2026-111943 in `crypto/x509`

## [v9.2.8](https://github.com/fayrus/registrator/releases/tag/v9.2.8) - 2026-10-18

### Security
- Bumped Go `1.26.5` → `1.27.2` to address the `golang/stdlib` CVEs reported in Go 1.27.1
- Upgraded `golang.org/x/net` v0.58.0 → v0.61.0 to address 5 HTTP/2 vulnerabilities reachable from registrator code (GO-2026-6603, GO-2026-6610, GO-2026-6611, GO-2026-6612, GO-2026-6617)
- `golang.org/x/sys` v0.47.0 → v0.49.0 and `golang.org/x/text` v0.41.0 → v0.43.0 updated transitively

## [v9.2.7](https://github.com/fayrus/registrator/releases/tag/v9.2.7) - 2026-09-27

### Fixed
- `etcd-legacy://` backend construction now uses a 5-second timeout for the `/version` check, so an unresponsive endpoint returns an error instead of blocking startup indefinitely
- The publish workflow now updates the Docker Hub description on every release; the previous README change check always failed silently on the shallow checkout

### Notes
- `etcd-legacy://` always connects over plain HTTP and does not support TLS; use `etcd://` if TLS is required

## [v9.2.6](https://github.com/fayrus/registrator/releases/tag/v9.2.6) - 2026-09-27

### Changed
- Pinned GitHub Actions runners to `ubuntu-26.04` instead of `ubuntu-latest` ahead of the Ubuntu 26.04 migration
- Updated all GitHub Actions to their latest releases, including `actions/checkout` v7, `actions/setup-go` v7 and `actions/setup-python` v7, and pinned Trivy to v0.74.0 in the publish security scan

## [v9.2.5](https://github.com/fayrus/registrator/releases/tag/v9.2.5) - 2026-09-27

### Fixed
- Upgraded `mkdocs-material` 9.7.6 → 9.7.7 in the documentation toolchain, resolving a DOM-based XSS in the `search.suggest` feature via the `q` URL parameter (GHSA-xvg9-69gf-fjrf / CVE-2026-73295)

## [v9.2.4](https://github.com/fayrus/registrator/releases/tag/v9.2.4) - 2026-09-16

### Fixed
- Upgraded `google.golang.org/grpc` v1.83.0 → v1.83.2, resolving a heap memory exhaustion DoS via HTTP/2 DATA frame fragmentation (CVE-2026-84304) and an xDS gRPC server panic on requests missing `:authority`/`Host` (CVE-2026-84445)

## [v9.2.3](https://github.com/fayrus/registrator/releases/tag/v9.2.3) - 2026-08-06

### Fixed
- `internal/etcdtls`: `Build()` now checks `AppendCertsFromPEM`'s return value and fails fast with a clear error when a CA file has no valid PEM certificates, instead of failing later and confusingly at TLS handshake time
- `registrator.go`: `connectWithRetry`'s loop no longer contains unreachable code; `validateArgs()` no longer returns a misleading always-nil `error` when it already terminates the process via `os.Exit` on invalid arguments
- Renamed `hostIp`/`Config.HostIp` to `hostIP`/`Config.HostIP` to follow Go initialism conventions

### Changed
- Hardened test assertions in `internal/etcdtls` and `internal/kvutil` (explicit error checks instead of discarding them, per-field assertions, nil-service checks on negative cases) following findings from GitHub AI code review and SonarCloud

## [v9.2.2](https://github.com/fayrus/registrator/releases/tag/v9.2.2) - 2026-08-06

### Fixed
- ZooKeeper `Register()` now updates the service znode with `Set()` instead of blindly calling `Create()` when it already exists, so periodic `-resync`/TTL refresh cycles no longer fail with a repeated "node exists" error for already-registered services

## [v9.2.1](https://github.com/fayrus/registrator/releases/tag/v9.2.1) - 2026-08-06

### Changed
- Bumped Go directive to `1.26.5` to match the `cgr.dev/chainguard/go:latest` base image
- Updated Go dependencies to their latest compatible versions via `go get -u` and `go mod tidy`, including `go.etcd.io/etcd/*` v3.6.10 → v3.7.1, `github.com/hashicorp/consul/api` v1.34.1 → v1.34.4, and `github.com/fsouza/go-dockerclient` v1.13.1 → v1.13.2
- Regenerated `docs/requirements.txt` to pull in `pymdown-extensions` 11.0.1

### Fixed
- Upgraded `google.golang.org/grpc` v1.80.0 → v1.83.0, resolving a high-severity xDS RBAC authorization bypass and HTTP/2 Rapid Reset DoS advisory (GHSA-hrxh-6v49-42gf)
- Upgraded `pymdown-extensions` to 11.0.1, resolving a path traversal in the `b64` extension that could disclose file contents (GHSA-9xwg-3r6f-jcx2)

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

[unreleased]: https://github.com/fayrus/registrator/compare/v9.2.18...HEAD
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
