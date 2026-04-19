# Security Review Checklist

Detailed rubric for security-reviewer. Each item is a yes/no check; "no" is a candidate finding, pending an attack scenario.

## Table of Contents

1. [Signature & Session Integrity](#1-signature--session-integrity)
2. [Input Validation at Boundaries](#2-input-validation-at-boundaries)
3. [Crypto Correctness](#3-crypto-correctness)
4. [Auth & AuthZ](#4-auth--authz)
5. [Secret / Key Handling](#5-secret--key-handling)
6. [DoS Surfaces](#6-dos-surfaces)
7. [TLS / Transport](#7-tls--transport)
8. [Logging Hygiene](#8-logging-hygiene)
9. [Economic Attack Surfaces](#9-economic-attack-surfaces)
10. [Concurrency Security](#10-concurrency-security)

---

## 1. Signature & Session Integrity

- [ ] Every RelayRequest received by a provider is verified (signature + session + pairing binding) before any work is done.
- [ ] Every RelayReply received by a consumer is verified (signature + session match + block finalization if required) before the result is returned to the end-user.
- [ ] Session IDs are unforgeable or at least unpredictable; a guess attack yields no access.
- [ ] Signatures are bound to a specific epoch / block height — replay across epochs is rejected.
- [ ] CU accounting can't be tricked by reordering (relay replay, out-of-order delivery).
- [ ] Smart-router mode: when signatures are skipped, the code path is clearly gated (no silent fallback on the rpcconsumer side).

**Watch for:** a `if skipVerify { ... }` branch that can be reached unintentionally; signature checks that depend only on fields the sender controls.

## 2. Input Validation at Boundaries

- [ ] HTTP/WS ingress enforces max request size per method/chain.
- [ ] JSON-RPC params are validated before being forwarded.
- [ ] Chain IDs / method names are allowlist-checked (not blocklist — blocklists are brittle).
- [ ] WS frame size and count-per-second limited per connection.
- [ ] Upstream response size bounded (no unbounded read of attacker-controlled body).
- [ ] gzip/brotli compression ratio bounded (decompression bomb).
- [ ] URL/host/port fields from config are sanity-checked (no SSRF via `http.Get(config.Upstream)`).
- [ ] Regex patterns compiled from config are length-bounded (ReDoS).
- [ ] No string interpolation into paths that then get `os.Open` (path traversal).

**Watch for:** `ioutil.ReadAll` (now `io.ReadAll`) on any body without a `io.LimitReader`.

## 3. Crypto Correctness

- [ ] Signature algorithm matches what verifiers expect (secp256k1 via `btcec`, ECDSA, etc.).
- [ ] Signatures are non-malleable (or the verifier tolerates both forms explicitly).
- [ ] Hashing before signing uses the expected algo; no "hash the hash" confusion.
- [ ] Randomness for nonce/session generation is `crypto/rand`, not `math/rand`.
- [ ] Constant-time compare (`subtle.ConstantTimeCompare`) for any secret equality check.
- [ ] Key derivation follows an agreed KDF where relevant.
- [ ] No homemade crypto — uses stdlib/btcec/curve25519/etc.

**Watch for:** `==` on `[]byte` containing MAC or session token; `math/rand` in a path that influences authentication.

## 4. Auth & AuthZ

- [ ] Consumer identity on provider side is validated against on-chain pairing (or static list for smart-router).
- [ ] Plan / subscription / project policy is enforced before expensive work.
- [ ] Badges / delegations verified on each relay (can't elevate via a stale badge).
- [ ] No TOCTOU: authorization decision and execution on the same snapshot.

## 5. Secret / Key Handling

- [ ] Private keys loaded from env / file / keyring (not hardcoded).
- [ ] Key material never in error strings, log lines, or panic traces.
- [ ] Keys not accidentally included in metrics labels or trace spans.
- [ ] In-memory keys zeroed after use where feasible (Go makes this hard; at minimum don't store longer than needed).
- [ ] Viper config precedence (env vs file) doesn't surprise the operator — default values don't mask an accidentally-missing key.
- [ ] `mask_consumer_logs` build flag, when enabled, actually masks the values it claims to; flag paths that bypass it.

## 6. DoS Surfaces

- [ ] No unbounded map that grows with attacker input (sessions per consumer, providers per chain, etc.).
- [ ] No unbounded channel buffer growing with offered load.
- [ ] No unbounded goroutine spawn per incoming request.
- [ ] Upstream stream handlers have read/write timeouts and a max body limit.
- [ ] Slow-body attacks (slowloris) mitigated via read deadlines.
- [ ] gRPC keepalive settings prevent resource pinning by idle adversaries.
- [ ] Cache stampede mitigated (singleflight, short-circuit on miss storm) where applicable.

**Watch for:** `chan T` without a `make(chan T, N)` justification; `go func() { ... }()` without a worker pool around it.

## 7. TLS / Transport

- [ ] Outbound TLS verifies server cert (unless dev/test flag explicitly disables).
- [ ] Cipher suite list excludes weak/deprecated (RC4, 3DES).
- [ ] Min TLS version 1.2 or higher.
- [ ] If mTLS is configured, cert rotation path works.
- [ ] gRPC services configured with reasonable keepalive and max stream sizes.

## 8. Logging Hygiene

- [ ] Private keys, session tokens, raw user payloads never logged at Info or above.
- [ ] Error wrapping doesn't accidentally include sensitive fields.
- [ ] Metrics labels are low-cardinality and don't include sensitive identifiers.
- [ ] Debug logs explicitly gated by a debug flag.

## 9. Economic Attack Surfaces

For provider-reward and relay-accounting paths:

- [ ] Proof of relay can't be reused (replay window bound).
- [ ] Proof includes consumer signature + provider counter-signature where appropriate.
- [ ] CU counter increments atomically; can't be rolled back via concurrent access.
- [ ] Conflict reports can't be forged (require valid signatures from both sides).
- [ ] Reward claim batching can't drop or duplicate individual proofs.
- [ ] Pairing reassignment mid-epoch doesn't enable free CU (ghost sessions).

## 10. Concurrency Security

- [ ] No TOCTOU between pairing check and relay execution.
- [ ] CU counter updates are atomic or mutex-protected.
- [ ] Session creation is idempotent under concurrent acquires with the same session ID.
- [ ] Cache writes don't race with invalidation (can lead to serving stale-authorized data).

---

## Reporting Pattern

When a checklist item fails:

1. Cite file:line.
2. Name the attacker persona.
3. Sketch the attack in 2–4 sentences (what they do, what they gain).
4. Rate severity per heuristics in SKILL.md.
5. Suggest concrete mitigation.

Example:

```markdown
### SEC-005 — RelayRequest session binding missing on retry path
- Severity: high
- Category: signature
- Location: `protocol/rpcprovider/reliability_manager.go:218`
- Description: On retry, the session ID is re-derived without re-verifying the original RelayRequest signature. ...
- Evidence: ...
- Attack Scenario: A malicious consumer observes a valid RelayRequest, crafts a replay with a different sessionID pointing at a victim session, submits via the retry endpoint. ...
- Impact: Victim's CU counter incremented; attacker receives relay for free.
- Recommendation: Re-verify signature on every retry, or sign the tuple (relay, session) explicitly.
- Confidence: medium — needs runtime trace to confirm sessionID re-derivation semantics.
```
