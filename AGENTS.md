# Repository Guidelines

## Project Structure
CLI entrypoints for `smartrouter` and `lavap` sit under `cmd/`. Core protocol implementation is in `protocol/`, with `rpcsmartrouter/` for the router server, `chainlib/` for chain-specific parsing and proxying, and `lavasession/` for connection management. Shared types are in `types/`, helpers in `utils/`, and chain specifications in `specs/`.

## Build, Test & Development Commands
- `make install-all` – install `smartrouter` and `lavap` binaries into `$GOBIN`.
- `make build-all` – build all binaries to `./build/`.
- `make test` – run `go test ./...` across all packages.
- `make test-short` – run smart router tests only.
- `make lint` – run `go vet ./...`.
- `make tidy` – run `go mod tidy`.

## Coding Style & Naming Conventions
Keep code `gofmt`/`goimports` clean (tabs for indent, spaces only for alignment). Use PascalCase for exported Go identifiers, snake_case for file names and directories. Keep packages single-purpose and minimize exported surface area.

## Testing Guidelines
Prefer table-driven tests named `TestComponent_Scenario`, colocated with the code in `protocol/` or `utils/`. Use `require`/`assert` helpers and always rerun `make test` before submitting.

## Commit & Pull Request Guidelines
Write short imperative commits with Conventional prefixes such as `feat:`, `fix(scope):`, or `chore:` and keep subjects under ~72 characters. PRs should describe impact, include test evidence, and confirm `make test` and `make lint` pass.

## Security & Configuration Tips
Never commit secrets. Sample config files (`smartrouter_lava.yml`) are templates only. Use environment variables for sensitive data like API keys and tokens.
