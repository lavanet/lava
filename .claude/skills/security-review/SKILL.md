---
name: security-review
description: "How the security-reviewer performs a security audit of the Lava protocol layer. Loaded when security-reviewer runs. Covers signing, session integrity, trust boundaries, input validation, crypto, DoS surfaces, and economic attack vectors on consumer/provider/smart-router."
---

# Security Review Skill

Method used by security-reviewer. Role at `.claude/agents/security-reviewer.md`; this file is the *how*.

## Method Overview

1. Read scope from `_workspace/00_input/scope.md`.
2. Load the threat-model context:
   - `.claude/skills/protocol-review/references/protocol-map.md` §§ "Trust Boundaries", "Request Path"
   - `references/checklist.md` of this skill
3. **Build a local threat model** — for the scope, list the boundaries crossed, the assets at stake, and the attacker personas.
4. Do a two-pass review:
   - **Pass 1 — Boundary walk:** Follow each trust boundary in scope. At each crossing, check that input is validated and output is signed/verified.
   - **Pass 2 — Cross-cutting:** Run the checklist (secret handling, crypto correctness, logging, DoS, economic).
5. For each issue, construct an explicit **attack scenario** before rating.
6. Emit `_workspace/01_security_findings.md`.

## Attacker Personas

Use these to frame findings:

| Persona | Capability | Interested in |
|---------|-----------|---------------|
| **End-user adversary** | Sends arbitrary HTTP/WS to consumer/smart-router | DoS the gateway, bypass auth, extract other users' data |
| **Malicious provider** | Signs relays, responds to consumer requests | Forge relays for reward, return malicious data, DoS consumers |
| **Malicious consumer** | Makes paired relays to a provider | Double-charge relays, replay, gain free CU, DoS the provider |
| **Compromised upstream** | Returns malicious JSON-RPC payloads | Trigger panics, fool finalization, exfil via response shape |
| **Passive network observer** | Reads traffic | Extract session keys, correlate users |

For each finding, specify which persona is relevant. If none, the issue is probably not security — re-file under another category or info.

## Severity Heuristics for This Reviewer

- **critical** — Practically exploitable now, with clear $ / data / uptime impact. Examples: unverified signature accepted on hot path; private key logged; unauthenticated reward claim.
- **high** — Conditionally exploitable (requires a specific attacker persona and some effort), with meaningful impact. Examples: replay within a session window; input validation gap leading to panic; memory exhaustion from unbounded map.
- **medium** — Defense-in-depth gap, not immediately exploitable but reduces margin. Examples: verbose error messages leaking internal state; TLS config allowing weak ciphers; missing timeouts on upstream.
- **low** — Hygiene issue with low exploitability. Examples: non-constant-time compare on a non-secret; debug logging of full request.
- **info** — Observation without action.

## Non-Findings

Do not file these — they are not security in this context:

- Generic Go "best practices" that don't tie to an attacker
- Style issues (STYLE owns those)
- Performance issues not tied to DoS
- On-chain logic concerns (out of scope — `x/` modules)

## Output

Per role template. Every finding needs an Attack Scenario section and an Impact section.

## Detailed Checklist

See `references/checklist.md`.
