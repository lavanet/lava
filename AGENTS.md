# Repository Guidelines

## Project Structure & Module Organization
Core Cosmos SDK logic lives in `app/`, while CLI entrypoints for `lavad`, `lavap`, and `lavavisor` sit under `cmd/`. Feature modules with keepers, params, and message types live in `x/<module>`. Shared networking flows are in `protocol/`, reusable helpers in `utils/` and `testutil/`, and protobuf specs plus schemas in `proto/` and `specs/`. Tests span unit files beside their packages, higher-level flows in `tests/` and `mock-node-server/`, and generated artifacts land in `build/`. Config resides in `config/`, and docker assets live in `docker/`.

## Build, Test & Development Commands
- `./scripts/init_install.sh` – one-time setup for Go toolchain and local deps.
- `LAVA_BINARY=all make install` – install all binaries into `$GOBIN`; `make build` writes to `./build`.
- `LAVA_BINARY=lavad make build` – rebuild a single binary; combine with `LAVA_BUILD_OPTIONS="static,release"` for reproducible outputs.
- `make test` – run `go test ./...`; `make lint` runs golangci-lint (formatting, gosimple, ineffassign, staticcheck).
- `make proto-gen` – regenerate protobuf stubs after editing `proto/` or `chain.schema.json`.
- `LAVA_BUILD_OPTIONS="static,release" make docker-build-all` – build multi-arch images for validator/provider deployments.

## Coding Style & Naming Conventions
Keep code `gofmt`/`goimports` clean (tabs for indent, spaces only for alignment). Follow `CODING_GUIDELINES.md`: keep packages single-purpose, depend on interfaces, and minimize exported surface area using `internal/` when access must be limited. Use PascalCase for exported Go identifiers and messages, snake_case for file names, protobuf fields, and directories. Treat every `make lint` warning as a blocker.

## Testing Guidelines
Prefer table-driven tests named `TestComponent_Scenario`, colocated with the code in `x/`, `protocol/`, or `utils/`. Use `require`/`assert` helpers, aim for ~70–80% coverage, and supplement with integration flows in `tests/` and `mock-node-server/`. Update `specs/*_test.md` with GIVEN–WHEN–THEN narratives and keep the subtree synced with `lavanet/lava-config` via `git subtree pull --prefix=specs ...`. Always rerun `make test` plus any targeted suites you touched (e.g., `go test ./x/pairing -run TestWeightedSelector`).

## Commit & Pull Request Guidelines
Write short imperative commits with Conventional prefixes such as `feat:`, `fix(scope):`, or `chore:` and keep subjects under ~72 characters; include issue references like `(#2108)` when helpful. PRs should link issues, describe validator/provider impact, call out updated configs or specs, and include CLI snippets or screenshots for UX changes. Confirm `make test`, `make lint`, and any docker or release commands touched, and document regenerated protobuf or schema artifacts.

## Security & Configuration Tips
Never commit provider or wallet secrets—sample manifests (`provider*_eth.yml`, `rpcprovider.yml`, `rpcconsumer.yml`) are templates only. Keep `config/*.yml` minimal, document any non-default ports or keys, and rely on environment variables for sensitive data. When editing protobuf- or schema-derived Go sources, run `make proto-gen` instead of hand-editing generated files.
