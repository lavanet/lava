# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Lava is a decentralized protocol for scalable blockchain data access, built on Cosmos SDK/CometBFT. It coordinates providers (serve blockchain APIs) and consumers (access blockchain data) through on-chain pairing, staking, and rewards. The native LAVA token incentivizes participation.

Three binaries: **lavad** (blockchain node/validator), **lavap** (protocol binary for providers/consumers/routers), **lavavisor** (process manager/upgrade tool).

## Build Commands

```bash
# Build all binaries to ./build/
LAVA_BINARY=all make build

# Install specific binary to $GOPATH/bin
LAVA_BINARY=lavad make install
LAVA_BINARY=lavap make install
LAVA_BINARY=lavavisor make install

# Static build (for containers)
LAVA_BUILD_OPTIONS="static" make build-all

# Release build
LAVA_BUILD_OPTIONS="static,release" make build-all
```

Build options: `static`, `release`, `nostrip`, `debug_mutex`, `mask_consumer_logs`.

## Testing

```bash
# Consensus/module unit tests
go test -v ./x/...

# Protocol tests (15m timeout recommended)
go test -v ./protocol/... -timeout 15m

# Utils tests
go test -v ./utils/...

# Run a single test
go test -run TestName ./path/to/package -timeout 15m

# E2E tests (20min timeout)
go test ./testutil/e2e/ -run ^TestLavaProtocol$ -timeout 1200s
go test ./testutil/e2e/ -run ^TestLavaProtocolPayment$ -timeout 1200s
```

## Linting

```bash
make lint
# Or directly: golangci-lint run --config .golangci.yml
```

Uses golangci-lint with gofumpt, goimports, govet, staticcheck, misspell, and others. See `.golangci.yml` for full config. Timeout is 7 minutes.

## Protobuf Code Generation

```bash
make proto-gen   # Uses Docker (ghcr.io/cosmos/proto-builder:0.13.5)
```

Proto definitions live in `proto/lavanet/lava/`. Requires buf, protoc-gen-gocosmos, protoc-gen-grpc-gateway. Run `./scripts/init_install.sh` to install proto toolchain.

## Architecture

### Two-Layer Design

**On-chain layer** (`x/` modules, `app/`): Cosmos SDK blockchain handling governance, pairing, staking, rewards, and specifications. The `app/app.go` file registers all modules.

**Off-chain layer** (`protocol/`): P2P relay protocol where consumers send signed gRPC requests to providers, who route them to upstream blockchain nodes and return signed responses.

### Cosmos Modules (`x/`)

| Module | Purpose |
|--------|---------|
| `spec` | API specification definitions for supported chains |
| `pairing` | Provider-consumer pairing algorithm per epoch |
| `subscription` | Consumer subscription management |
| `plans` | Subscription plan definitions |
| `projects` | Project management within subscriptions |
| `epochstorage` | Snapshot storage per epoch |
| `fixationstore` | Reference-counted differential storage |
| `timerstore` | Block-time based callback scheduling |
| `dualstaking` | Dual validator/provider staking |
| `rewards` | Provider reward distribution |
| `conflict` | Dispute resolution between conflicting provider responses |
| `downtime` | Validator downtime detection |
| `protocol` | Protocol version governance |

Each module follows the Cosmos SDK keeper pattern with its own message handlers, queries, genesis state, and parameter governance.

### Protocol Layer (`protocol/`)

| Package | Purpose |
|---------|---------|
| `rpcconsumer` | Decentralized RPC gateway (uses on-chain pairing) |
| `rpcsmartrouter` | Centralized RPC gateway (static provider config) |
| `rpcprovider` | Serves blockchain data to consumers |
| `chainlib` | Multi-chain support (JSON-RPC, REST, gRPC, TendermintRPC, WebSocket) |
| `lavasession` | Consumer-provider session management |
| `provideroptimizer` | QoS-based provider selection using Weighted Random Selection (WRS) |
| `statetracker` | Tracks blockchain state updates (pairings, epochs, specs) |
| `chaintracker` | Tracks latest block heights on served chains |
| `relaycore` | Core relay state machine and processing logic |
| `lavaprotocol` | Protocol abstractions (retry management, finalization) |
| `metrics` | Prometheus metrics collection |

### Request Flow

Consumer receives user HTTP request → StateTracker queries Lava chain for pairing → ProviderOptimizer selects best provider(s) by QoS → RelayCore sends signed gRPC relay → Provider routes to upstream node via ChainLib → Provider signs and returns response → Consumer validates and returns to user.

### Key Frameworks and Dependencies

- **Cosmos SDK v0.47.13** (custom lavanet fork) + **CometBFT v0.37.5** for blockchain
- **gRPC/Protobuf** for P2P relay protocol and blockchain queries
- **Cobra/Viper** for CLI and configuration
- **go-ethereum v1.14.13** for Ethereum integration
- **Prometheus** for metrics, **Pyroscope** for profiling
- Go version: **1.23**

## Harness: Protocol Code Development & Review

**Goal:** Structured, multi-angle review (architecture / security / performance / style) and design-first development for the Lava **off-chain protocol layer** (`protocol/**` — rpcconsumer, rpcprovider, rpcsmartrouter, chainlib, lavasession, relaycore, statetracker, chaintracker, provideroptimizer, lavaprotocol, qos, metrics). **Excludes** on-chain `x/` Cosmos modules.

**Agent team:**

| Agent | Role |
|-------|------|
| architecture-reviewer | Module boundaries, dependency direction, interface design, concurrency architecture, state management, error model, request-path shape, testability |
| security-reviewer | Signing & session integrity, trust boundaries, input validation, crypto correctness, auth, secret handling, DoS, TLS, logging hygiene, economic attack surfaces |
| performance-reviewer | Hot-path allocations, goroutine lifecycle, locking, channels, I/O patterns, serialization, caching, memory footprint, GC pressure |
| style-reviewer | golangci-lint compliance (`.golangci.yml`), naming, error idioms (`utils.LavaFormatError`), comments, package layout, test style, nil handling, type assertions, codebase consistency |
| synthesis-reporter | Merges 4 reviewer outputs into a single prioritized report: deduplicates, reconciles severity, clusters themes, estimates effort |
| protocol-architect | Pre-implementation design doc (problem, goals, current state, API, data flow, concurrency, failure modes, compatibility, tests, alternatives) |
| protocol-implementer | Go implementation following the design; matches codebase voice; lint + build + race clean |
| protocol-tester | Unit / integration / e2e tests; table-driven; race detector; deterministic time; fakes over mocks |

**Skills:**

| Skill | Purpose | Used by |
|-------|---------|---------|
| `protocol-review` | Orchestrator: parallel fan-out → synthesis review flow | orchestrates all 4 reviewers + synthesis-reporter |
| `protocol-develop` | Orchestrator: design → implement → test pipeline (optional post-review) | orchestrates protocol-architect → protocol-implementer → protocol-tester |
| `architecture-review` | Methodology: structural checklist per package & topic | architecture-reviewer |
| `security-review` | Methodology: threat-model + 10-section security checklist | security-reviewer |
| `performance-review` | Methodology: hot/warm/cold classification + perf checklist | performance-reviewer |
| `style-review` | Methodology: `.golangci.yml`-grounded style checklist, clustering rules | style-reviewer |
| `review-synthesis` | Methodology: deduplication, severity reconciliation, prioritization, clustering | synthesis-reporter |
| `protocol-design` | Methodology: exploration budget, design priorities, alternatives, failure enumeration | protocol-architect |
| `protocol-implementation` | Methodology: codebase-voice matching, error wrapping, concurrency discipline, lint/test | protocol-implementer |
| `protocol-testing` | Methodology: layer choice, table-driven, race, determinism, benchmarks | protocol-tester |

**Execution rules:**

- **Review request** (audit, review, inspect, evaluate a PR / package / diff): invoke `protocol-review` skill — it fans out to the 4 reviewers in parallel with `model: "opus"` and synthesizes results. Report lands at `REVIEW_REPORT.md` (repo root) and `_workspace/02_synthesis_report.md`.
- **Development request** (design + implement + test a change): invoke `protocol-develop` skill — it pipelines architect → implementer → tester; optionally auto-triggers `protocol-review` for post-implementation audit when the change touches signing paths, `lavasession`, `rewardserver`, or >3 packages.
- **Follow-up / re-invocation** (update design, re-review with feedback, fix specific findings): both orchestrators support partial re-runs via `_workspace/00_input/feedback.md`. IDs (ARCH-NNN, SEC-NNN, etc.) are stable across runs.
- **Simple questions and small edits:** do NOT invoke the harness. The harness is for substantive changes or audits, not for fixing a typo or answering a single question.
- **All agents run with `model: "opus"`** — the harness depends on deep reasoning.
- **Scope boundary:** the harness explicitly **excludes** on-chain `x/` Cosmos modules, `app/`, consensus code. For those, use standard tools without the harness.
- **Audit trail:** intermediate artifacts land in `_workspace/` and are preserved (not deleted) for post-hoc review.

**Directory structure:**

```
.claude/
├── agents/
│   ├── architecture-reviewer.md
│   ├── security-reviewer.md
│   ├── performance-reviewer.md
│   ├── style-reviewer.md
│   ├── synthesis-reporter.md
│   ├── protocol-architect.md
│   ├── protocol-implementer.md
│   └── protocol-tester.md
└── skills/
    ├── protocol-review/
    │   ├── SKILL.md
    │   └── references/
    │       ├── protocol-map.md        # shared: package topography
    │       ├── review-taxonomy.md     # shared: severity / category / ID / schema
    │       └── report-template.md     # synthesis output template
    ├── protocol-develop/
    │   ├── SKILL.md
    │   └── references/
    │       └── workflow.md
    ├── architecture-review/
    │   ├── SKILL.md
    │   └── references/checklist.md
    ├── security-review/
    │   ├── SKILL.md
    │   └── references/checklist.md
    ├── performance-review/
    │   ├── SKILL.md
    │   └── references/checklist.md
    ├── style-review/
    │   ├── SKILL.md
    │   └── references/checklist.md
    ├── review-synthesis/
    │   └── SKILL.md
    ├── protocol-design/
    │   └── SKILL.md
    ├── protocol-implementation/
    │   └── SKILL.md
    └── protocol-testing/
        └── SKILL.md
```

**Workspace layout (runtime, not committed by default):**

```
_workspace/
├── 00_input/
│   ├── scope.md          # for review runs
│   ├── requirements.md   # for dev runs
│   └── feedback.md       # for re-invocation
├── 01_architecture_findings.md  (review)
├── 01_security_findings.md      (review)
├── 01_performance_findings.md   (review)
├── 01_style_findings.md         (review)
├── 01_design.md                 (dev)
├── 02_synthesis_report.md       (review)
├── 02_implementation.md         (dev)
├── 03_testing.md                (dev)
└── 04_post_review/              (dev + review chain)
```

**Change log:**

| Date | Change | Target | Rationale |
|------|--------|--------|-----------|
| 2026-04-16 | Initial harness construction | all | User requested comprehensive code development + parallel-review harness scoped to protocol layer (excluding x/ consensus) |

