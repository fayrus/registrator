# Contributing to Registrator

Thanks for your interest in Registrator. Bug fixes and documentation improvements are welcome.

For anything larger than a small fix, please open an issue first so we can agree on the approach before you spend time on it.

Never include secret values, tokens, or credentials in issues, pull requests, logs, or test fixtures. To report a vulnerability, follow the [security policy](SECURITY.md) instead of opening a public issue.

## Getting started

You need Go 1.27.4 and golangci-lint v2.15.0, the versions CI uses. With [mise](https://mise.jdx.dev/), `mise install` installs them and the other pinned tools from `mise.toml`.

```bash
make test    # run the test suite with the race detector
make lint    # run golangci-lint
make local   # build the Docker image locally
```

The same checks are available as mise tasks: `mise run test`, `mise run lint`, `mise run vulncheck`, and `mise run scan` (builds the image and scans it with Trivy).

## Making a change

1. Create a branch from `main` named `<type>/<what-it-does>`, for example `fix/reject-empty-keys`. Types are `feat`, `fix`, `chore`, `docs`, and `refactor`.
2. Keep the change focused on one topic. For bug fixes, start with a test that fails without your fix.
3. Update the docs if you change behavior or configuration, and describe the effect for users in `CHANGELOG.md`.
4. Use commit titles like `fix: reject empty keys`.
5. Open a pull request against `main` and fill in the template. CI must pass before merging.

## Versions and releases

Merging to `main` publishes a release with the version in the `VERSION` file, unless the pull request only touches `.github/`, `docs/`, `Makefile`, `mise.toml`, `LICENSE`, `.dockerignore`, Markdown files in the repository root, `sonar-project.properties`, or `requirements-docs.txt`.

Any other pull request needs a new version, so that a published release is not overwritten; the publish workflow fails if the version already exists. If you are not sure which version to use, leave it as it is and a maintainer will take care of it before merging.

## Code of conduct

By taking part in this project, you agree to follow its [Code of Conduct](CODE_OF_CONDUCT.md).
