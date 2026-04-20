# Change Log

All notable changes to this project will be documented in this file.

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

[unreleased]: https://github.com/fayrus/registrator/compare/v8.0.0...HEAD
[v8.0.0]: https://github.com/fayrus/registrator/releases/tag/v8.0.0
