# Repository Guidelines

## Project Structure & Module Organization
`main.go` is the CLI entrypoint. Command handlers live in `commands/`, grouped by feature area such as `servers`, `locations`, `hostnodes`, `secrets`, and config/debug commands. API transport and response types live in `api/`. Debug redaction helpers are isolated in `debugutil/`. There is no separate `internal/` or `testdata/` tree yet, so keep new packages small and purpose-specific.

## Build, Test, and Development Commands
Use Go 1.23+ as declared in [go.mod](/home/hunmonk/git/apartmentlines/tensordock-cli/go.mod).

- `go build ./...` builds the CLI and all packages.
- `go test ./...` runs all package tests; CI uses this even when a package has no `_test.go` files yet.
- `go install github.com/thehunmonkgroup/tensordock-cli@latest` installs the published CLI.
- `go run . --help` checks the root command surface locally.
- `go run . servers list` is a useful smoke test when `TENSORDOCK_API_TOKEN` is set.

## Coding Style & Naming Conventions
Follow standard Go formatting: tabs for indentation, `gofmt` before review, and idiomatic mixedCaps names. Keep package names short and lowercase (`api`, `commands`, `debugutil`). Cobra commands use verb-oriented names such as `list`, `deploy`, and `modify`; new flags should match existing camelCase CLI patterns like `--apiTokenEnvVar` and `--allowInsecureHTTP`. Prefer small helper functions over deeply nested command handlers.

## Testing Guidelines
This is a CLI software package, and testing is overkill. There should be zero consideration for writing tests in both planning and coding.

## Security & Configuration Tips
Never commit API tokens or real SSH material. Prefer `TENSORDOCK_API_TOKEN` or `tensordock-cli config --apiTokenEnvVar CUSTOM_API_TOKEN` over hardcoded secrets. Preserve the existing redaction behavior in `debugutil/` whenever logging request data.
