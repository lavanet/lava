#!/usr/bin/env python3
"""
WRS end-to-end analyzer (framework-friendly)

What it does:
  0) Runs `test3_simple.sh` (optional --no-run) OR analyzes externally-provided artifacts.
  1) Computes theoretical provider composite scores + expected distribution
     from the input configs (gap_blocks / availability / delay_ms / stake),
     spec average_block_time, and the provided weights.
  2) Parses the run artifacts (provider header hits + consumer log) and
     prints diffs between theoretical and real distributions.

This is intended for local/dev usage (it shells out to bash and uses the repo scripts).
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import math
import os
import re
import subprocess
import sys
import bisect
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Tuple


# Repo root (this file lives in wrs_tests/_framework/)
REPO_ROOT = Path(__file__).resolve().parents[2]
TEST3_SH = REPO_ROOT / "test3_simple.sh"
LOGS_DIR = REPO_ROOT / "testutil" / "debugging" / "logs"
CONSUMER_LOG = LOGS_DIR / "CONSUMER.test3_simple.log"
PROVIDER_HITS = LOGS_DIR / "provider_hits.test3_simple.txt"


@dataclass(frozen=True)
class ProviderTheory:
    name: str
    gap_blocks: int
    availability: float
    delay_ms: int
    stake_ratio: float
    # computed
    raw_latency_sec: float
    raw_sync_sec: float
    normalized_latency: float
    normalized_sync: float
    normalized_availability: float
    normalized_stake: float
    composite: float
    expected_prob: float


@dataclass(frozen=True)
class Weights:
    availability: float
    latency: float
    sync: float
    stake: float


@dataclass(frozen=True)
class ProviderRealSummary:
    name: str
    samples: int
    # raw EWMA-resolved values as logged
    raw_availability: float
    raw_latency_sec: float
    raw_sync_sec: float
    raw_stake: str
    # normalized values as logged
    normalized_availability: float
    normalized_latency: float
    normalized_sync: float
    normalized_stake: float
    composite: float
    observed_prob: float


def parse_external_health_events(consumer_log_text: str) -> List[Tuple[dt.datetime, str]]:
    """
    Parse external relay success events for the `health` method.
    Returns list of (timestamp, provider) in log order.
    """
    year = dt.datetime.now().year
    pat = re.compile(r"\bRelay succeeded\b.*\bmethod=health\b.*\bprovider=([^\s]+)")
    events: List[Tuple[dt.datetime, str]] = []
    for line in consumer_log_text.splitlines():
        m = pat.search(line)
        if not m:
            continue
        ts = _parse_log_timestamp(line, year)
        if not ts:
            continue
        events.append((ts, m.group(1)))
    return events


def match_external_events_to_breakdowns(
    events: List[Tuple[dt.datetime, str]],
    breakdowns: Dict[str, List[dict]],
    max_skew_seconds: float = 2.0,
) -> Dict[str, List[dict]]:
    """
    For each external (ts, provider) event, pick the nearest breakdown sample (before OR after)
    within max_skew_seconds, and attach it to the event timestamp (so time-series bucketing follows
    external relays, not internal scoring log timing).
    """
    prov_recs: Dict[str, List[dict]] = {}
    prov_ts: Dict[str, List[dt.datetime]] = {}
    for p, recs in breakdowns.items():
        rr = [r for r in recs if isinstance(r.get("ts"), dt.datetime)]
        rr.sort(key=lambda r: r["ts"])
        prov_recs[p] = rr
        prov_ts[p] = [r["ts"] for r in rr]

    out: Dict[str, List[dict]] = {}
    for ets, provider in sorted(events, key=lambda x: x[0]):
        ts_list = prov_ts.get(provider, [])
        rec_list = prov_recs.get(provider, [])
        if not ts_list:
            continue
        j = bisect.bisect_left(ts_list, ets)
        candidates: List[dict] = []
        if j - 1 >= 0:
            candidates.append(rec_list[j - 1])
        if j < len(rec_list):
            candidates.append(rec_list[j])
        best = None
        best_dt = None
        for r in candidates:
            d = abs((ets - r["ts"]).total_seconds())
            if d <= max_skew_seconds and (best_dt is None or d < best_dt):
                best = r
                best_dt = d
        if best is None:
            continue
        # copy record but force ts to external event timestamp (so buckets don't go empty)
        r2 = dict(best)
        r2["ts"] = ets
        out.setdefault(provider, []).append(r2)
    return out


def write_theory_vs_real_table_md(
    out_path: Path,
    theory: List[ProviderTheory],
    expected_probs: Dict[str, float],
    real: Dict[str, ProviderRealSummary],
) -> None:
    providers = sorted({t.name for t in theory})
    tmap = {t.name: t for t in theory}
    lines: List[str] = []
    lines.append("# WRS: Theory vs Real\n")
    lines.append("| provider | exp_prob% | obs_prob% | norm_lat (t/r) | norm_sync (t/r) | norm_av (t/r) | norm_stake (t/r) | composite (t/r) | samples |")
    lines.append("|---|---:|---:|---:|---:|---:|---:|---:|---:|")
    for p in providers:
        t = tmap.get(p)
        r = real.get(p)
        if not t or not r:
            continue
        lines.append(
            f"| `{p}` | {expected_probs.get(p,0.0)*100:.3f} | {r.observed_prob*100:.3f} | "
            f"{t.normalized_latency:.6f} / {r.normalized_latency:.6f} | "
            f"{t.normalized_sync:.6f} / {r.normalized_sync:.6f} | "
            f"{t.normalized_availability:.6f} / {r.normalized_availability:.6f} | "
            f"{t.normalized_stake:.6f} / {r.normalized_stake:.6f} | "
            f"{t.composite:.6f} / {r.composite:.6f} | {r.samples} |"
        )
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def die(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8", errors="replace")
    except FileNotFoundError:
        die(f"missing file: {path}")


def run_test3(warmup: int, measured: int, parallelism: int, ignore_cache: bool) -> None:
    if not TEST3_SH.exists():
        die(f"missing {TEST3_SH}")

    env = dict(os.environ)
    env["WARMUP_REQUESTS"] = str(warmup)
    env["NUM_REQUESTS"] = str(measured)
    env["PARALLELISM"] = str(parallelism)
    env["IGNORE_CACHE"] = "true" if ignore_cache else "false"

    print(
        f"Running {TEST3_SH} with WARMUP_REQUESTS={warmup} NUM_REQUESTS={measured} "
        f"PARALLELISM={parallelism} IGNORE_CACHE={env['IGNORE_CACHE']}"
    )
    subprocess.run(["bash", str(TEST3_SH)], cwd=str(REPO_ROOT), env=env, check=True)


def parse_weights_from_test3_sh(contents: str) -> Weights:
    # Pull weights from the consumer command line in test3_simple.sh
    def get_float(flag: str) -> float:
        m = re.search(rf"{re.escape(flag)}\s+([0-9.]+)", contents)
        if not m:
            die(f"could not find {flag} in test3_simple.sh")
        return float(m.group(1))

    return Weights(
        sync=get_float("--provider-optimizer-sync-weight"),
        availability=get_float("--provider-optimizer-availability-weight"),
        latency=get_float("--provider-optimizer-latency-weight"),
        stake=get_float("--provider-optimizer-stake-weight"),
    )


def parse_provider_test_response_files(contents: str) -> List[Tuple[Path, Path]]:
    """
    Returns list of (provider_config_path, test_responses_json_path) from the script.
    """
    pairs: List[Tuple[Path, Path]] = []
    # Example:
    # lavap rpcprovider config/provider_examples/provider1_test3...yml --test_mode --test_responses ./wrs_test_configs/test3_synced.json ...
    pat = re.compile(
        r"lavap\s+rpcprovider\s+(\S+)\s+--test_mode\s+--test_responses\s+(\S+)\s",
        re.IGNORECASE,
    )
    for m in pat.finditer(contents):
        provider_cfg = (REPO_ROOT / m.group(1)).resolve()
        responses = (REPO_ROOT / m.group(2)).resolve()
        pairs.append((provider_cfg, responses))
    if not pairs:
        die("could not find provider --test_responses entries in test3_simple.sh")
    return pairs


def parse_provider_name_from_provider_cfg(yml_text: str) -> str:
    # minimal parse: look for "name: provider-2220-..."
    m = re.search(r"(?m)^\s*-\s*name:\s*([^\s]+)\s*$", yml_text)
    if not m:
        # fallback: "name:" without dash
        m = re.search(r"(?m)^\s*name:\s*([^\s]+)\s*$", yml_text)
    if not m:
        die("could not parse provider name from provider yml")
    return m.group(1).strip()


def parse_average_block_time_ms_from_lava_spec(spec_json_path: Path, chain_index: str = "LAV1") -> int:
    data = json.loads(read_text(spec_json_path))
    # structure: {"proposal": {"specs": [ { "index": "LAV1", "average_block_time": 15000, ... }, ... ]}}
    specs = data.get("proposal", {}).get("specs", [])
    for s in specs:
        if s.get("index") == chain_index and "average_block_time" in s:
            return int(s["average_block_time"])
    die(f"could not find average_block_time for index={chain_index} in {spec_json_path}")


def clamp(x: float, lo: float, hi: float) -> float:
    return lo if x < lo else hi if x > hi else x


def weighted_percentile(values: List[float], weights: List[float], q: float) -> float:
    """
    Deterministic weighted percentile for a discrete distribution.
    Returns the smallest value v such that cumulativeWeight(values<=v) >= q * totalWeight.
    """
    if not values or not weights or len(values) != len(weights):
        return 0.0
    if q <= 0:
        return min(values)
    if q >= 1:
        return max(values)
    pairs = sorted(zip(values, weights), key=lambda x: x[0])
    total = sum(w for _, w in pairs)
    if total <= 0:
        return min(values)
    target = q * total
    c = 0.0
    for v, w in pairs:
        c += max(0.0, float(w))
        if c >= target:
            return float(v)
    return float(pairs[-1][0])


def normalize_availability(av: float) -> float:
    # From score.MinAcceptableAvailability (0.80) and weighted_selector.normalizeAvailability
    min_acceptable = 0.80
    if av < min_acceptable:
        return 0.0
    return clamp((av - min_acceptable) / (1.0 - min_acceptable), 0.0, 1.0)


def normalize_latency_fixed(latency_sec: float) -> float:
    # Phase 1 fallback: WorstLatencyScore = 30 seconds.
    max_latency = 30.0
    if latency_sec <= 0:
        return 1.0
    return max(0.0, 1.0 - (latency_sec / max_latency))


def normalize_latency_adaptive(latency_sec: float, p10: float, p90: float) -> float:
    # From weighted_selector.normalizeLatency Phase2 (P10-P90) with Phase1 fallback.
    if not (p10 > 0 and p90 > 0 and p90 > p10) or any(map(lambda v: math.isnan(v) or math.isinf(v), [p10, p90])):
        return normalize_latency_fixed(latency_sec)
    clamped = clamp(latency_sec, p10, p90)
    norm = 1.0 - (clamped - p10) / (p90 - p10)
    return clamp(norm, 0.0, 1.0)


def apply_strategy_adjustments(latency: float, sync: float, strategy: str) -> Tuple[float, float]:
    """
    Mirrors `weighted_selector.go` applyStrategyAdjustments for the normalized (0-1) latency/sync scores.
    """
    if strategy == "latency":
        if latency > 0.7:
            latency = math.pow(latency, 0.8)
    elif strategy == "sync-freshness":
        if sync > 0.7:
            sync = math.pow(sync, 0.8)
    elif strategy in ("accuracy", "distributed"):
        latency = math.pow(latency, 1.2)
        sync = math.pow(sync, 1.2)
    # cost/privacy/balanced -> no adjustments
    return latency, sync


def normalize_stake_equal(n: int) -> Tuple[float, float]:
    # stakeScore = sqrt(stake/totalStake). With equal stakes ratio=1/n.
    if n <= 0:
        return 0.0, 0.0
    ratio = 1.0 / float(n)
    return ratio, math.sqrt(ratio)


def normalize_sync_adaptive(sync_lag_sec: float, p10: float, p90: float) -> float:
    # From weighted_selector.normalizeSync Phase2
    if not (p10 > 0 and p90 > 0 and p90 > p10) or any(map(lambda v: math.isnan(v) or math.isinf(v), [p10, p90])):
        # Phase1 fallback (WorstSyncScore = 1200s)
        max_sync = 1200.0
        if sync_lag_sec <= 0:
            return 1.0
        return max(0.0, 1.0 - (sync_lag_sec / max_sync))
    clamped = clamp(sync_lag_sec, p10, p90)
    norm = 1.0 - (clamped - p10) / (p90 - p10)
    return clamp(norm, 0.0, 1.0)


def compute_theory(
    providers: List[Tuple[str, dict]],
    weights: Weights,
    average_block_time_ms: int,
    stake_values: List[int] | None = None,
    strategy: str = "balanced",
    min_selection_chance: float = 0.01,
) -> Tuple[List[ProviderTheory], Dict[str, float]]:
    """
    providers: list of (provider_name, test_responses_json dict)
    Returns (per-provider theory list, expected probability map).
    """
    avg_block_time_sec = average_block_time_ms / 1000.0

    # Build raw sync lags (steady-state) from gap_blocks.
    gaps = []
    for _, cfg in providers:
        gaps.append(int(cfg.get("gap_blocks", 0)))

    # Steady-state sync lag: gap_blocks * avg_block_time_sec
    raw_syncs = [g * avg_block_time_sec for g in gaps]

    # Latency adaptive bounds (Phase 2) and Sync adaptive bounds (Phase 2):
    # In production these come from a global (across providers) decaying t-digest.
    #
    # For *theory*, we compute P10/P90 from the *theoretical global distribution* of samples:
    # - Each provider contributes its steady-state raw metric value.
    # - The mixture weights are the providers' theoretical selection probabilities.
    #
    # Because selection probabilities depend on the normalization (and thus on P10/P90),
    # we solve this as a small fixed-point iteration.
    raw_latencies = []
    for _, cfg in providers:
        responses = cfg.get("responses", {})
        health = responses.get("health", {}) if isinstance(responses, dict) else {}
        delay_ms = int(health.get("delay_ms", 0))
        raw_latencies.append(max(0.0, delay_ms / 1000.0))
    # Initialize bounds with equal provider weights (or simple min/max if empty).
    init_w = [1.0] * len(raw_latencies)
    lat_p10 = weighted_percentile(raw_latencies, init_w, 0.10)
    lat_p90 = weighted_percentile(raw_latencies, init_w, 0.90)
    # Clamp to safety bounds from score_config / doc:
    lat_p10 = clamp(lat_p10, 0.001, 10.0)
    lat_p90 = clamp(lat_p90, 1.0, 30.0)
    if lat_p90 <= lat_p10:
        lat_p90 = max(lat_p10 + 0.001, 1.0)

    sync_p10 = weighted_percentile(raw_syncs, [1.0] * len(raw_syncs), 0.10)
    sync_p90 = weighted_percentile(raw_syncs, [1.0] * len(raw_syncs), 0.90)
    sync_p10 = clamp(sync_p10, 0.1, 60.0)
    sync_p90 = clamp(sync_p90, 30.0, 1200.0)
    if sync_p90 <= sync_p10:
        sync_p90 = max(sync_p10 + 1.0, 30.0)

    nprov = len(providers)
    # Match weighted_selector: normalize weights to sum=1 if user provides a different sum.
    wsum = weights.availability + weights.latency + weights.sync + weights.stake
    if wsum > 0 and abs(wsum - 1.0) > 0.001:
        weights = Weights(
            availability=weights.availability / wsum,
            latency=weights.latency / wsum,
            sync=weights.sync / wsum,
            stake=weights.stake / wsum,
        )
    # Stake normalization:
    # - If explicit stake values are provided, use stakeScore = sqrt(stake/totalStake)
    # - Otherwise fallback to equal-stake assumption (sqrt(1/N))
    stake_ratios: List[float] = []
    stake_scores: List[float] = []
    if stake_values is not None and len(stake_values) > 0:
        if len(stake_values) != nprov:
            die(f"stake values count ({len(stake_values)}) must match providers count ({nprov})")
        total = float(sum(max(0, int(v)) for v in stake_values))
        if total <= 0:
            # fallback to equal
            stake_ratio, norm_stake = normalize_stake_equal(nprov)
            stake_ratios = [stake_ratio] * nprov
            stake_scores = [norm_stake] * nprov
        else:
            for v in stake_values:
                vv = float(max(0, int(v)))
                r = vv / total
                if r > 1.0:
                    r = 1.0
                stake_ratios.append(r)
                stake_scores.append(math.sqrt(r) if r > 0 else 0.0)
    else:
        stake_ratio, norm_stake = normalize_stake_equal(nprov)
        stake_ratios = [stake_ratio] * nprov
        stake_scores = [norm_stake] * nprov

    def _compute_once(current_lat_p10: float, current_lat_p90: float, current_sync_p10: float, current_sync_p90: float) -> Tuple[List[ProviderTheory], Dict[str, float]]:
        theory_local: List[ProviderTheory] = []
        composites_local: Dict[str, float] = {}

        for idx, ((name, cfg), gap, raw_sync) in enumerate(zip(providers, gaps, raw_syncs)):
            responses = cfg.get("responses", {})
            health = responses.get("health", {}) if isinstance(responses, dict) else {}
            availability = float(health.get("availability", 1.0))
            delay_ms = int(health.get("delay_ms", 0))
            raw_latency_sec = delay_ms / 1000.0

            norm_av = normalize_availability(availability)
            norm_latency = normalize_latency_adaptive(raw_latency_sec, current_lat_p10, current_lat_p90)
            norm_sync = normalize_sync_adaptive(raw_sync, current_sync_p10, current_sync_p90)
            norm_latency, norm_sync = apply_strategy_adjustments(norm_latency, norm_sync, strategy=strategy)

            composite = (
                weights.availability * norm_av
                + weights.sync * norm_sync
                + weights.latency * norm_latency
                + weights.stake * stake_scores[idx]
            )
            if composite < min_selection_chance:
                composite = min_selection_chance
            if composite > 1.0:
                composite = 1.0

            composites_local[name] = composite
            theory_local.append(
                ProviderTheory(
                    name=name,
                    gap_blocks=gap,
                    availability=availability,
                    delay_ms=delay_ms,
                    stake_ratio=stake_ratios[idx],
                    raw_sync_sec=raw_sync,
                    raw_latency_sec=raw_latency_sec,
                    normalized_latency=norm_latency,
                    normalized_sync=norm_sync,
                    normalized_availability=norm_av,
                    normalized_stake=stake_scores[idx],
                    composite=composite,
                    expected_prob=0.0,
                )
            )

        total_local = sum(composites_local.values())
        if total_local <= 0:
            die("theoretical composite sum is <= 0; cannot compute probabilities")
        expected_local = {name: composites_local[name] / total_local for name in composites_local}
        theory_local = [
            ProviderTheory(**{**t.__dict__, "expected_prob": expected_local.get(t.name, 0.0)})
            for t in theory_local
        ]
        return theory_local, expected_local

    # Fixed-point: update bounds from theoretical selection distribution.
    theory, expected_probs = _compute_once(lat_p10, lat_p90, sync_p10, sync_p90)
    for _ in range(25):
        # Build mixture weights in provider order
        mix_w = [expected_probs.get(name, 0.0) for name, _ in providers]
        new_lat_p10 = weighted_percentile(raw_latencies, mix_w, 0.10)
        new_lat_p90 = weighted_percentile(raw_latencies, mix_w, 0.90)
        new_lat_p10 = clamp(new_lat_p10, 0.001, 10.0)
        new_lat_p90 = clamp(new_lat_p90, 1.0, 30.0)
        if new_lat_p90 <= new_lat_p10:
            new_lat_p90 = max(new_lat_p10 + 0.001, 1.0)

        new_sync_p10 = weighted_percentile(raw_syncs, mix_w, 0.10)
        new_sync_p90 = weighted_percentile(raw_syncs, mix_w, 0.90)
        new_sync_p10 = clamp(new_sync_p10, 0.1, 60.0)
        new_sync_p90 = clamp(new_sync_p90, 30.0, 1200.0)
        if new_sync_p90 <= new_sync_p10:
            new_sync_p90 = max(new_sync_p10 + 1.0, 30.0)

        # Convergence check
        if (
            abs(new_lat_p10 - lat_p10) < 1e-6
            and abs(new_lat_p90 - lat_p90) < 1e-6
            and abs(new_sync_p10 - sync_p10) < 1e-6
            and abs(new_sync_p90 - sync_p90) < 1e-6
        ):
            break
        lat_p10, lat_p90, sync_p10, sync_p90 = new_lat_p10, new_lat_p90, new_sync_p10, new_sync_p90
        theory, expected_probs = _compute_once(lat_p10, lat_p90, sync_p10, sync_p90)

    return theory, expected_probs


def _parse_log_timestamp(line: str, year: int) -> dt.datetime | None:
    # Consumer logs start like: "Feb 10 10:59:50 ..."
    # We parse the first 15 chars as "%b %d %H:%M:%S"
    if len(line) < 15:
        return None
    prefix = line[:15]
    try:
        parsed = dt.datetime.strptime(prefix, "%b %d %H:%M:%S")
        return parsed.replace(year=year)
    except Exception:
        return None


def find_measured_time_window_from_consumer_log(consumer_log_text: str, warmup: int, measured: int) -> Tuple[dt.datetime, dt.datetime]:
    """
    Use method=health relay-succeeded lines to bracket the measured window.
    Assumes warmup+measured external requests were made in sequence.
    """
    year = dt.datetime.now().year
    pat = re.compile(r"\bRelay succeeded\b.*\bmethod=health\b")
    times: List[dt.datetime] = []
    for line in consumer_log_text.splitlines():
        if not pat.search(line):
            continue
        ts = _parse_log_timestamp(line, year)
        if ts:
            times.append(ts)

    total_needed = warmup + measured
    if len(times) < total_needed:
        die(f"consumer log has only {len(times)} health relays, expected at least {total_needed} (warmup+measured)")

    # take the last warmup+measured to be robust to older runs appended
    times = times[-total_needed:]
    start = times[warmup]  # first measured request timestamp
    end = times[-1]        # last measured request timestamp
    return start, end


def parse_real_score_breakdowns(
    consumer_log_text: str,
    window_start: dt.datetime,
    window_end: dt.datetime,
    slack_seconds: int = 2,
) -> Dict[str, List[dict]]:
    """
    Parse 'Provider score calculation breakdown' lines and return per-provider samples
    within [window_start - slack, window_end + slack].
    """
    year = dt.datetime.now().year
    start = window_start - dt.timedelta(seconds=slack_seconds)
    end = window_end + dt.timedelta(seconds=slack_seconds)

    samples: Dict[str, List[dict]] = {}
    if "Provider score calculation breakdown" not in consumer_log_text:
        return samples

    for line in consumer_log_text.splitlines():
        if "Provider score calculation breakdown" not in line:
            continue
        ts = _parse_log_timestamp(line, year)
        if not ts or ts < start or ts > end:
            continue

        tokens = line.strip().split()
        kv: Dict[str, str] = {}
        for tok in tokens:
            if "=" in tok:
                k, v = tok.split("=", 1)
                kv[k] = v

        provider = kv.get("provider")
        if not provider:
            continue

        def f(key: str, default: float = 0.0) -> float:
            try:
                return float(kv.get(key, default))
            except Exception:
                return default

        rec = {
            "ts": ts,
            "raw_availability": f("raw_availability"),
            "raw_latency_sec": f("raw_latency_sec"),
            "raw_sync_sec": f("raw_sync_sec"),
            "raw_stake": kv.get("raw_stake", ""),
            "normalized_availability": f("normalized_availability"),
            "normalized_latency": f("normalized_latency"),
            "normalized_sync": f("normalized_sync"),
            "normalized_stake": f("normalized_stake"),
            "composite_score": f("composite_score"),
        }
        samples.setdefault(provider, []).append(rec)

    return samples


def summarize_real_scores(samples: Dict[str, List[dict]], observed: Counter) -> Dict[str, ProviderRealSummary]:
    total = sum(observed.values()) or 1

    def mean(vals: List[float]) -> float:
        return sum(vals) / len(vals) if vals else 0.0

    out: Dict[str, ProviderRealSummary] = {}
    for provider, recs in samples.items():
        if not recs:
            continue
        out[provider] = ProviderRealSummary(
            name=provider,
            samples=len(recs),
            raw_availability=mean([r["raw_availability"] for r in recs]),
            raw_latency_sec=mean([r["raw_latency_sec"] for r in recs]),
            raw_sync_sec=mean([r["raw_sync_sec"] for r in recs]),
            raw_stake=recs[-1].get("raw_stake", ""),
            normalized_availability=mean([r["normalized_availability"] for r in recs]),
            normalized_latency=mean([r["normalized_latency"] for r in recs]),
            normalized_sync=mean([r["normalized_sync"] for r in recs]),
            normalized_stake=mean([r["normalized_stake"] for r in recs]),
            composite=mean([r["composite_score"] for r in recs]),
            observed_prob=(observed.get(provider, 0) / total),
        )
    return out


def _svg_escape(s: str) -> str:
    return (
        s.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
        .replace("'", "&#39;")
    )


def write_svg_grouped_bars(
    out_path: Path,
    title: str,
    providers: List[str],
    series: List[Tuple[str, List[float]]],
    y_max: float = 1.0,
) -> None:
    """
    Minimal dependency-free grouped bar chart writer (SVG).
    Values should be 0..y_max.
    """
    width = 1200
    height = 520
    margin = 60
    plot_w = width - 2 * margin
    plot_h = height - 2 * margin

    nprov = len(providers)
    nser = len(series)
    if nprov == 0 or nser == 0:
        return

    group_w = plot_w / nprov
    bar_w = group_w / (nser + 1)
    colors = ["#4C78A8", "#F58518", "#54A24B", "#E45756", "#72B7B2", "#B279A2"]

    def y(v: float) -> float:
        vv = clamp(v, 0.0, y_max)
        return margin + (1.0 - (vv / y_max)) * plot_h

    lines: List[str] = []
    lines.append(f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}">')
    lines.append(f'<rect x="0" y="0" width="{width}" height="{height}" fill="white"/>')
    lines.append(f'<text x="{margin}" y="{margin-25}" font-family="Arial" font-size="18" font-weight="bold">{_svg_escape(title)}</text>')

    # axes
    lines.append(f'<line x1="{margin}" y1="{margin}" x2="{margin}" y2="{margin+plot_h}" stroke="#333" stroke-width="1"/>')
    lines.append(f'<line x1="{margin}" y1="{margin+plot_h}" x2="{margin+plot_w}" y2="{margin+plot_h}" stroke="#333" stroke-width="1"/>')

    # y ticks
    for t in [0.0, 0.25, 0.5, 0.75, 1.0]:
        yy = y(t * y_max)
        lines.append(f'<line x1="{margin-5}" y1="{yy}" x2="{margin}" y2="{yy}" stroke="#333" stroke-width="1"/>')
        lines.append(f'<text x="{margin-10}" y="{yy+4}" font-family="Arial" font-size="12" text-anchor="end">{t:.2f}</text>')

    # bars + labels
    for i, prov in enumerate(providers):
        gx = margin + i * group_w
        lines.append(f'<text x="{gx + group_w/2}" y="{margin+plot_h+30}" font-family="Arial" font-size="12" text-anchor="middle">{_svg_escape(prov)}</text>')
        for j, (sname, values) in enumerate(series):
            v = values[i]
            bx = gx + (j + 0.5) * bar_w
            by = y(v)
            bh = (margin + plot_h) - by
            color = colors[j % len(colors)]
            lines.append(f'<rect x="{bx}" y="{by}" width="{bar_w*0.9}" height="{bh}" fill="{color}"/>')
            lines.append(f'<text x="{bx + bar_w*0.45}" y="{by-4}" font-family="Arial" font-size="10" text-anchor="middle">{v:.3f}</text>')

    # legend
    lx = margin + plot_w - 260
    ly = margin - 40
    for idx, (sname, _) in enumerate(series):
        cy = ly + idx * 16
        color = colors[idx % len(colors)]
        lines.append(f'<rect x="{lx}" y="{cy}" width="12" height="12" fill="{color}"/>')
        lines.append(f'<text x="{lx+18}" y="{cy+11}" font-family="Arial" font-size="12">{_svg_escape(sname)}</text>')

    lines.append("</svg>")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines), encoding="utf-8")


def write_svg_multi_line(
    out_path: Path,
    title: str,
    x_values: List[float],
    series: List[Tuple[str, List[float]]],
    y_max: float = 1.0,
    y_min: float = 0.0,
    x_label: str = "t (sec)",
    y_label: str = "score",
) -> None:
    """
    Minimal dependency-free multi-line chart writer (SVG).
    x_values are floats (e.g., seconds since start). Each series must match len(x_values).
    """
    width = 1200
    height = 420
    margin_l = 70
    margin_r = 30
    margin_t = 60
    margin_b = 55
    plot_w = width - margin_l - margin_r
    plot_h = height - margin_t - margin_b

    if not x_values or not series:
        return

    n = len(x_values)
    for _, ys in series:
        if len(ys) != n:
            die("write_svg_multi_line: series length mismatch")

    x0, x1 = min(x_values), max(x_values)
    if x1 == x0:
        x1 = x0 + 1.0
    if y_max == y_min:
        y_max = y_min + 1.0

    def x_map(x: float) -> float:
        return margin_l + (x - x0) * plot_w / (x1 - x0)

    def y_map(y: float) -> float:
        yy = clamp(y, y_min, y_max)
        return margin_t + (1.0 - (yy - y_min) / (y_max - y_min)) * plot_h

    colors = ["#4C78A8", "#F58518", "#54A24B", "#E45756", "#72B7B2", "#B279A2"]

    lines: List[str] = []
    lines.append(f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}">')
    lines.append(f'<rect x="0" y="0" width="{width}" height="{height}" fill="white"/>')
    lines.append(f'<text x="{margin_l}" y="{margin_t-25}" font-family="Arial" font-size="18" font-weight="bold">{_svg_escape(title)}</text>')

    # axes
    lines.append(f'<line x1="{margin_l}" y1="{margin_t}" x2="{margin_l}" y2="{margin_t+plot_h}" stroke="#333" stroke-width="1"/>')
    lines.append(f'<line x1="{margin_l}" y1="{margin_t+plot_h}" x2="{margin_l+plot_w}" y2="{margin_t+plot_h}" stroke="#333" stroke-width="1"/>')

    # axis labels
    lines.append(f'<text x="{margin_l+plot_w/2}" y="{margin_t+plot_h+45}" font-family="Arial" font-size="12" text-anchor="middle">{_svg_escape(x_label)}</text>')
    lines.append(f'<text x="{20}" y="{margin_t+plot_h/2}" font-family="Arial" font-size="12" text-anchor="middle" transform="rotate(-90 20 {margin_t+plot_h/2})">{_svg_escape(y_label)}</text>')

    # y ticks + grid
    for t in [0.0, 0.25, 0.5, 0.75, 1.0]:
        yyv = y_min + t * (y_max - y_min)
        yy = y_map(yyv)
        lines.append(f'<line x1="{margin_l-5}" y1="{yy}" x2="{margin_l}" y2="{yy}" stroke="#333" stroke-width="1"/>')
        lines.append(f'<text x="{margin_l-10}" y="{yy+4}" font-family="Arial" font-size="12" text-anchor="end">{yyv:.2f}</text>')
        lines.append(f'<line x1="{margin_l}" y1="{yy}" x2="{margin_l+plot_w}" y2="{yy}" stroke="#ddd" stroke-width="1"/>')

    # x ticks (up to 10)
    tick_count = min(10, n)
    if tick_count >= 2:
        for i in range(tick_count):
            idx = int(i * (n - 1) / (tick_count - 1))
            xv = x_values[idx]
            xx = x_map(xv)
            lines.append(f'<line x1="{xx}" y1="{margin_t+plot_h}" x2="{xx}" y2="{margin_t+plot_h+5}" stroke="#333" stroke-width="1"/>')
            lines.append(f'<text x="{xx}" y="{margin_t+plot_h+20}" font-family="Arial" font-size="12" text-anchor="middle">{xv:.0f}</text>')

    # series lines
    for si, (name, ys) in enumerate(series):
        color = colors[si % len(colors)]
        pts = [f"{x_map(x_values[i]):.2f},{y_map(ys[i]):.2f}" for i in range(n)]
        lines.append(f'<polyline fill="none" stroke="{color}" stroke-width="2" points="{" ".join(pts)}"/>')

    # legend
    lx = margin_l + plot_w - 260
    ly = margin_t - 40
    for idx, (sname, _) in enumerate(series):
        cy = ly + idx * 16
        color = colors[idx % len(colors)]
        lines.append(f'<line x1="{lx}" y1="{cy+6}" x2="{lx+18}" y2="{cy+6}" stroke="{color}" stroke-width="3"/>')
        lines.append(f'<text x="{lx+24}" y="{cy+10}" font-family="Arial" font-size="12">{_svg_escape(sname)}</text>')

    lines.append("</svg>")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines), encoding="utf-8")


def write_svg_multi_line_custom(
    out_path: Path,
    title: str,
    x_values: List[float],
    series: List[Tuple[str, List[float], str, str | None]],
    y_max: float = 1.0,
    y_min: float = 0.0,
    x_label: str = "t (sec)",
    y_label: str = "score",
) -> None:
    """
    Same as write_svg_multi_line but allows per-series color and optional dash style.
    series: (name, y_values, color, dasharray)
      - dasharray example: "6,4" or None for solid.
    """
    width = 1200
    height = 420
    margin_l = 70
    margin_r = 30
    margin_t = 60
    margin_b = 55
    plot_w = width - margin_l - margin_r
    plot_h = height - margin_t - margin_b

    if not x_values or not series:
        return

    n = len(x_values)
    for _, ys, _, _ in series:
        if len(ys) != n:
            die("write_svg_multi_line_custom: series length mismatch")

    x0, x1 = min(x_values), max(x_values)
    if x1 == x0:
        x1 = x0 + 1.0
    if y_max == y_min:
        y_max = y_min + 1.0

    def x_map(x: float) -> float:
        return margin_l + (x - x0) * plot_w / (x1 - x0)

    def y_map(y: float) -> float:
        yy = clamp(y, y_min, y_max)
        return margin_t + (1.0 - (yy - y_min) / (y_max - y_min)) * plot_h

    lines: List[str] = []
    lines.append(f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}">')
    lines.append(f'<rect x="0" y="0" width="{width}" height="{height}" fill="white"/>')
    lines.append(f'<text x="{margin_l}" y="{margin_t-25}" font-family="Arial" font-size="18" font-weight="bold">{_svg_escape(title)}</text>')

    # axes
    lines.append(f'<line x1="{margin_l}" y1="{margin_t}" x2="{margin_l}" y2="{margin_t+plot_h}" stroke="#333" stroke-width="1"/>')
    lines.append(f'<line x1="{margin_l}" y1="{margin_t+plot_h}" x2="{margin_l+plot_w}" y2="{margin_t+plot_h}" stroke="#333" stroke-width="1"/>')

    # axis labels
    lines.append(f'<text x="{margin_l+plot_w/2}" y="{margin_t+plot_h+45}" font-family="Arial" font-size="12" text-anchor="middle">{_svg_escape(x_label)}</text>')
    lines.append(f'<text x="{20}" y="{margin_t+plot_h/2}" font-family="Arial" font-size="12" text-anchor="middle" transform="rotate(-90 20 {margin_t+plot_h/2})">{_svg_escape(y_label)}</text>')

    # y ticks + grid
    for t in [0.0, 0.25, 0.5, 0.75, 1.0]:
        yyv = y_min + t * (y_max - y_min)
        yy = y_map(yyv)
        lines.append(f'<line x1="{margin_l-5}" y1="{yy}" x2="{margin_l}" y2="{yy}" stroke="#333" stroke-width="1"/>')
        lines.append(f'<text x="{margin_l-10}" y="{yy+4}" font-family="Arial" font-size="12" text-anchor="end">{yyv:.2f}</text>')
        lines.append(f'<line x1="{margin_l}" y1="{yy}" x2="{margin_l+plot_w}" y2="{yy}" stroke="#ddd" stroke-width="1"/>')

    # x ticks (up to 10)
    tick_count = min(10, n)
    if tick_count >= 2:
        for i in range(tick_count):
            idx = int(i * (n - 1) / (tick_count - 1))
            xv = x_values[idx]
            xx = x_map(xv)
            lines.append(f'<line x1="{xx}" y1="{margin_t+plot_h}" x2="{xx}" y2="{margin_t+plot_h+5}" stroke="#333" stroke-width="1"/>')
            lines.append(f'<text x="{xx}" y="{margin_t+plot_h+20}" font-family="Arial" font-size="12" text-anchor="middle">{xv:.0f}</text>')

    # series
    for name, ys, color, dash in series:
        pts = [f"{x_map(x_values[i]):.2f},{y_map(ys[i]):.2f}" for i in range(n)]
        dash_attr = f' stroke-dasharray="{dash}"' if dash else ""
        lines.append(f'<polyline fill="none" stroke="{color}" stroke-width="2"{dash_attr} points="{" ".join(pts)}"/>')

    # legend (wrap-ish)
    lx = margin_l + plot_w - 320
    ly = margin_t - 40
    for idx, (sname, _, color, dash) in enumerate(series):
        cy = ly + idx * 16
        dash_attr = f' stroke-dasharray="{dash}"' if dash else ""
        lines.append(f'<line x1="{lx}" y1="{cy+6}" x2="{lx+18}" y2="{cy+6}" stroke="{color}" stroke-width="3"{dash_attr}/>')
        lines.append(f'<text x="{lx+24}" y="{cy+10}" font-family="Arial" font-size="12">{_svg_escape(sname)}</text>')

    lines.append("</svg>")
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines), encoding="utf-8")


def combine_svgs_vertical(out_path: Path, svg_paths: List[Path], padding: int = 20) -> None:
    """
    Combine multiple same-width SVGs into one tall SVG by stacking them vertically.
    Assumes each input SVG has explicit width/height attributes (as generated by this script).
    """
    svgs: List[Tuple[int, int, str]] = []
    for p in svg_paths:
        txt = read_text(p)
        m = re.search(r'<svg[^>]*\bwidth="(\d+)"[^>]*\bheight="(\d+)"[^>]*>', txt)
        if not m:
            die(f"could not parse width/height from svg: {p}")
        w, h = int(m.group(1)), int(m.group(2))
        # extract inner content
        start = txt.find(">")
        end = txt.rfind("</svg>")
        if start == -1 or end == -1 or end <= start:
            die(f"could not extract inner svg content: {p}")
        inner = txt[start + 1 : end].strip()
        svgs.append((w, h, inner))

    if not svgs:
        return

    width = svgs[0][0]
    for w, _, _ in svgs:
        if w != width:
            die("combine_svgs_vertical requires all SVGs to have the same width")

    total_height = sum(h for _, h, _ in svgs) + padding * (len(svgs) - 1)
    y = 0
    out_lines: List[str] = []
    out_lines.append(f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{total_height}">')
    out_lines.append(f'<rect x="0" y="0" width="{width}" height="{total_height}" fill="white"/>')
    for _, h, inner in svgs:
        out_lines.append(f'<g transform="translate(0,{y})">')
        out_lines.append(inner)
        out_lines.append("</g>")
        y += h + padding
    out_lines.append("</svg>")

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(out_lines), encoding="utf-8")


def write_plots(
    plots_dir: Path,
    theory: List[ProviderTheory],
    expected_probs: Dict[str, float],
    real: Dict[str, ProviderRealSummary],
    observed: Counter,
) -> None:
    providers = sorted([t.name for t in theory])
    tmap = {t.name: t for t in theory}

    write_svg_grouped_bars(
        plots_dir / "theoretical_scores.svg",
        "Theoretical normalized scores + composite (steady-state)",
        providers,
        [
            ("availability_norm", [tmap[p].normalized_availability for p in providers]),
            ("latency_norm", [tmap[p].normalized_latency for p in providers]),
            ("sync_norm", [tmap[p].normalized_sync for p in providers]),
            ("stake_norm", [tmap[p].normalized_stake for p in providers]),
            ("composite", [tmap[p].composite for p in providers]),
        ],
        y_max=1.0,
    )

    total = sum(observed.values()) or 1
    write_svg_grouped_bars(
        plots_dir / "distribution_expected_vs_observed.svg",
        "Expected vs Observed provider selection probability",
        providers,
        [
            ("expected_prob", [expected_probs.get(p, 0.0) for p in providers]),
            ("observed_prob", [observed.get(p, 0) / total for p in providers]),
        ],
        y_max=1.0,
    )

    if real:
        # default empty summary for missing provider keys
        def empty(p: str) -> ProviderRealSummary:
            return ProviderRealSummary(
                name=p,
                samples=0,
                raw_availability=0.0,
                raw_latency_sec=0.0,
                raw_sync_sec=0.0,
                raw_stake="",
                normalized_availability=0.0,
                normalized_latency=0.0,
                normalized_sync=0.0,
                normalized_stake=0.0,
                composite=0.0,
                observed_prob=(observed.get(p, 0) / total),
            )

        write_svg_grouped_bars(
            plots_dir / "real_scores_from_logs.svg",
            "Real normalized scores + composite (from consumer log breakdown lines)",
            providers,
            [
                ("availability_norm", [real.get(p, empty(p)).normalized_availability for p in providers]),
                ("latency_norm", [real.get(p, empty(p)).normalized_latency for p in providers]),
                ("sync_norm", [real.get(p, empty(p)).normalized_sync for p in providers]),
                ("stake_norm", [real.get(p, empty(p)).normalized_stake for p in providers]),
                ("composite", [real.get(p, empty(p)).composite for p in providers]),
            ],
            y_max=1.0,
        )

    # Combined dashboard
    combine_svgs_vertical(
        plots_dir / "wrs_dashboard.svg",
        [
            plots_dir / "theoretical_scores.svg",
            plots_dir / "real_scores_from_logs.svg",
            plots_dir / "distribution_expected_vs_observed.svg",
        ],
        padding=30,
    )


def write_timeseries_plots(
    plots_dir: Path,
    consumer_text: str,
    window_start: dt.datetime,
    window_end: dt.datetime,
    bucket_sec: int,
    providers: List[str],
    samples: Dict[str, List[dict]],
    theory: List[ProviderTheory] | None = None,
    expected_probs: Dict[str, float] | None = None,
) -> None:
    """
    Time-series graphs:
      - per-provider normalized metric scores + composite over time (bucket_sec buckets, avg within bucket)
      - selection distribution over time (bucket_sec buckets, probability per bucket)
    """
    start = window_start.replace(microsecond=0)
    end = window_end.replace(microsecond=0)
    nb = int(((end - start).total_seconds()) // bucket_sec) + 1
    x_vals = [i * bucket_sec for i in range(nb)]

    def bucketize(provider: str, key: str) -> List[float]:
        vals = [0.0] * nb
        cnt = [0] * nb
        for rec in samples.get(provider, []):
            ts: dt.datetime = rec["ts"]
            idx = int((ts - start).total_seconds() // bucket_sec)
            if 0 <= idx < nb:
                try:
                    v = float(rec.get(key, 0.0))
                except Exception:
                    v = 0.0
                vals[idx] += v
                cnt[idx] += 1
        for i in range(nb):
            if cnt[i]:
                vals[i] = vals[i] / cnt[i]
            else:
                # Avoid misleading drops to 0 when a time bucket has no samples.
                # Carry forward the previous value (if any).
                vals[i] = vals[i - 1] if i > 0 else 0.0
        return vals

    metrics = [
        ("availability_norm", "normalized_availability"),
        ("latency_norm", "normalized_latency"),
        ("sync_norm", "normalized_sync"),
        ("stake_norm", "normalized_stake"),
        ("composite", "composite_score"),
    ]

    for title, key in metrics:
        series = [(p, bucketize(p, key)) for p in providers]
        write_svg_multi_line(
            plots_dir / f"timeseries_{title}.svg",
            f"{title} over time (bucket={bucket_sec}s, avg within bucket)",
            x_vals,
            series,
            y_min=0.0,
            y_max=1.0,
            x_label="seconds since first measured request",
            y_label=title,
        )

    dx, probs = parse_measured_health_timeseries(consumer_text, window_start, window_end, bucket_sec=bucket_sec)
    if dx and probs:
        series = [(p, probs.get(p, [0.0] * len(dx))) for p in providers]
        write_svg_multi_line(
            plots_dir / "timeseries_distribution.svg",
            f"selection distribution over time (bucket={bucket_sec}s)",
            dx,
            series,
            y_min=0.0,
            y_max=1.0,
            x_label="seconds since first measured request",
            y_label="selection probability",
        )

    combine_svgs_vertical(
        plots_dir / "wrs_timeseries_dashboard.svg",
        [plots_dir / f"timeseries_{m[0]}.svg" for m in metrics] + [plots_dir / "timeseries_distribution.svg"],
        padding=25,
    )

    # Comparison plots: theoretical (steady-state constants) vs real, on the same chart.
    if theory is not None:
        tmap = {t.name: t for t in theory}
        palette = ["#4C78A8", "#F58518", "#54A24B", "#E45756", "#72B7B2", "#B279A2"]
        color_map = {p: palette[i % len(palette)] for i, p in enumerate(providers)}

        def t_const(p: str, metric_key: str) -> float:
            t = tmap.get(p)
            if not t:
                return 0.0
            if metric_key == "normalized_availability":
                return t.normalized_availability
            if metric_key == "normalized_latency":
                return t.normalized_latency
            if metric_key == "normalized_sync":
                return t.normalized_sync
            if metric_key == "normalized_stake":
                return t.normalized_stake
            if metric_key == "composite_score":
                return t.composite
            return 0.0

        for title, key in metrics:
            series_custom: List[Tuple[str, List[float], str, str | None]] = []
            for p in providers:
                c = color_map.get(p, "#333333")
                real_y = bucketize(p, key)
                th_y = [t_const(p, key)] * nb
                series_custom.append((f"{p} real", real_y, c, None))
                series_custom.append((f"{p} theoretical", th_y, c, "6,4"))

            write_svg_multi_line_custom(
                plots_dir / f"timeseries_compare_{title}.svg",
                f"{title}: theoretical (dashed) vs real (solid), bucket={bucket_sec}s",
                x_vals,
                series_custom,
                y_min=0.0,
                y_max=1.0,
                x_label="seconds since first measured request",
                y_label=title,
            )

        # Distribution chart: show REAL distribution only (no theoretical overlay).
        # The expected distribution is still printed in the console report + written to theory_vs_real.md.
        if dx and probs:
            series_custom = []
            for p in providers:
                c = color_map.get(p, "#333333")
                real_y = probs.get(p, [0.0] * len(dx))
                series_custom.append((f"{p} real", real_y, c, None))

            write_svg_multi_line_custom(
                plots_dir / "timeseries_compare_distribution.svg",
                f"distribution (real only), bucket={bucket_sec}s",
                dx,
                series_custom,
                y_min=0.0,
                y_max=1.0,
                x_label="seconds since first measured request",
                y_label="selection probability",
            )

        # Combined comparison dashboard
        combine_svgs_vertical(
            plots_dir / "wrs_timeseries_compare_dashboard.svg",
            [plots_dir / f"timeseries_compare_{m[0]}.svg" for m in metrics] + [plots_dir / "timeseries_compare_distribution.svg"],
            padding=25,
        )

def parse_observed_from_provider_hits(path: Path) -> Counter:
    if not path.exists():
        die(f"missing provider hits file: {path} (did the script run?)")
    lines = [ln.strip() for ln in read_text(path).splitlines() if ln.strip()]
    return Counter(lines)


def parse_observed_from_consumer_log_health(path: Path) -> Counter:
    if not path.exists():
        return Counter()
    # e.g. Relay succeeded ... method=health provider=provider-2220...
    pat = re.compile(r"\bmethod=health\b.*\bprovider=([^\s]+)")
    c = Counter()
    for line in read_text(path).splitlines():
        m = pat.search(line)
        if m:
            c[m.group(1)] += 1
    return c


def parse_measured_health_timeseries(
    consumer_log_text: str,
    window_start: dt.datetime,
    window_end: dt.datetime,
    bucket_sec: int = 1,
) -> Tuple[List[float], Dict[str, List[float]]]:
    """
    Build per-second selection probability time-series from 'Relay succeeded method=health provider=...'.
    Returns (x_values_seconds_since_start, provider->probabilities_per_bucket).
    """
    year = dt.datetime.now().year
    pat = re.compile(r"\bRelay succeeded\b.*\bmethod=health\b.*\bprovider=([^\s]+)")
    events: List[Tuple[dt.datetime, str]] = []
    for line in consumer_log_text.splitlines():
        m = pat.search(line)
        if not m:
            continue
        ts = _parse_log_timestamp(line, year)
        if not ts or ts < window_start or ts > window_end:
            continue
        events.append((ts, m.group(1)))

    if not events:
        return [], {}

    start = window_start.replace(microsecond=0)
    end = window_end.replace(microsecond=0)
    nb = int(((end - start).total_seconds()) // bucket_sec) + 1
    providers = sorted({p for _, p in events})
    counts = {p: [0] * nb for p in providers}
    total = [0] * nb

    for ts, p in events:
        idx = int((ts - start).total_seconds() // bucket_sec)
        if 0 <= idx < nb:
            counts[p][idx] += 1
            total[idx] += 1

    probs: Dict[str, List[float]] = {}
    for p in providers:
        probs[p] = [(counts[p][i] / total[i]) if total[i] else 0.0 for i in range(nb)]

    # Forward-fill buckets with no events to avoid rendering misleading zeros.
    for i in range(nb):
        if total[i] == 0:
            for p in providers:
                probs[p][i] = probs[p][i - 1] if i > 0 else 0.0

    x_vals = [i * bucket_sec for i in range(nb)]
    return x_vals, probs


def print_report(
    measured_n: int,
    weights: Weights,
    avg_block_time_ms: int,
    theory: List[ProviderTheory],
    expected_probs: Dict[str, float],
    observed: Counter,
    observed_health_log: Counter,
    real_summary: Dict[str, ProviderRealSummary] | None = None,
) -> None:
    print("\n=== Inputs ===")
    print(f"average_block_time_ms: {avg_block_time_ms} (={avg_block_time_ms/1000.0:.3f}s)")
    print(f"weights: availability={weights.availability} sync={weights.sync} latency={weights.latency} stake={weights.stake}")

    print("\n=== Theoretical (steady-state) ===")
    for t in sorted(theory, key=lambda x: x.name):
        print(
            f"- {t.name}: gap_blocks={t.gap_blocks} raw_latency_sec={t.raw_latency_sec:.3f} raw_sync_sec={t.raw_sync_sec:.3f} "
            f"norm_av={t.normalized_availability:.6f} norm_lat={t.normalized_latency:.6f} norm_sync={t.normalized_sync:.6f} norm_stake={t.normalized_stake:.6f} "
            f"composite={t.composite:.6f} expected_prob={expected_probs[t.name]*100:.3f}% expected_count={expected_probs[t.name]*measured_n:.2f}"
        )

    print("\n=== Observed (from response headers) ===")
    obs_total = sum(observed.values())
    print(f"total={obs_total}")
    for name, cnt in observed.most_common():
        print(f"- {name}: count={cnt} prob={cnt/obs_total*100:.3f}%")

    if observed_health_log:
        print("\n=== Observed (from consumer log: Relay succeeded method=health) ===")
        log_total = sum(observed_health_log.values())
        print(f"total={log_total}")
        for name, cnt in observed_health_log.most_common():
            print(f"- {name}: count={cnt} prob={cnt/log_total*100:.3f}%")

    # Diff
    print("\n=== Diff (headers observed vs theoretical) ===")
    names = sorted(expected_probs.keys())
    for name in names:
        exp_p = expected_probs[name]
        obs_p = (observed.get(name, 0) / obs_total) if obs_total else 0.0
        exp_c = exp_p * obs_total
        obs_c = observed.get(name, 0)
        print(f"- {name}: expected={exp_p*100:.3f}% ({exp_c:.2f}) observed={obs_p*100:.3f}% ({obs_c}) delta={((obs_p-exp_p)*100):+.3f}% ({(obs_c-exp_c):+.2f})")

    # Chi-square (rough)
    if obs_total:
        chi2 = 0.0
        for name in names:
            exp = expected_probs[name] * obs_total
            if exp > 0:
                chi2 += (observed.get(name, 0) - exp) ** 2 / exp
        print(f"\nchi2 (df={len(names)-1}): {chi2:.3f}")

    if real_summary:
        print("\n=== Theory vs Real (from consumer log breakdown averages) ===")
        tmap = {t.name: t for t in theory}
        for name in sorted(tmap.keys()):
            t = tmap[name]
            r = real_summary.get(name)
            if not r:
                continue
            print(
                f"- {name}: "
                f"norm_lat theory={t.normalized_latency:.6f} real={r.normalized_latency:.6f} | "
                f"norm_sync theory={t.normalized_sync:.6f} real={r.normalized_sync:.6f} | "
                f"norm_av theory={t.normalized_availability:.6f} real={r.normalized_availability:.6f} | "
                f"norm_stake theory={t.normalized_stake:.6f} real={r.normalized_stake:.6f} | "
                f"composite theory={t.composite:.6f} real={r.composite:.6f} (samples={r.samples})"
            )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--no-run", action="store_true", help="do not run test3_simple.sh; only analyze existing logs")
    ap.add_argument("--warmup", type=int, default=50, help="warmup requests (env WARMUP_REQUESTS)")
    ap.add_argument("--measured", type=int, default=500, help="measured requests (env NUM_REQUESTS)")
    ap.add_argument("--parallel", type=int, default=1, help="measured request parallelism (env PARALLELISM)")
    # Optional overrides to analyze arbitrary test directories (instead of hard-coded test3_simple.sh + default logs dir)
    ap.add_argument("--consumer-log", type=str, default="", help="path to consumer log to analyze (defaults to repo testutil/debugging/logs/CONSUMER.test3_simple.log)")
    ap.add_argument("--provider-hits", type=str, default="", help="path to provider hits file (defaults to repo testutil/debugging/logs/provider_hits.test3_simple.txt)")
    ap.add_argument("--provider-json", action="append", default=[], help="provider test-mode JSON file(s) used for theoretical calculation (repeatable, order matters)")
    ap.add_argument("--provider-name", action="append", default=[], help="provider name(s) matching --provider-json files (repeatable, same count). If omitted, defaults to provider-2220/2221/2222.")
    ap.add_argument("--provider-stake", action="append", default=[], help="provider stake values (ulava) matching --provider-json files (repeatable, same count). If omitted, assumes equal stake.")
    ap.add_argument("--availability-weight", type=float, default=float("nan"), help="override availability weight (if set, overrides script-parsed weights)")
    ap.add_argument("--latency-weight", type=float, default=float("nan"), help="override latency weight (if set, overrides script-parsed weights)")
    ap.add_argument("--sync-weight", type=float, default=float("nan"), help="override sync weight (if set, overrides script-parsed weights)")
    ap.add_argument("--stake-weight", type=float, default=float("nan"), help="override stake weight (if set, overrides script-parsed weights)")
    ap.add_argument(
        "--strategy",
        type=str,
        default="balanced",
        choices=["balanced", "latency", "sync-freshness", "accuracy", "distributed", "cost", "privacy"],
        help="strategy adjustments to apply to normalized latency/sync scores (default: balanced/no-op)",
    )
    ap.add_argument("--min-selection-chance", type=float, default=0.01, help="minimum composite score clamp (default: 0.01)")
    ap.add_argument(
        "--exclude-non-external-relays",
        action="store_true",
        default=True,
        help="compute 'real' score averages only from external method=health relays (default: true)",
    )
    ap.add_argument(
        "--include-non-external-relays",
        action="store_true",
        help="include all score-breakdown lines in 'real' averages (disables --exclude-non-external-relays)",
    )
    ap.add_argument("--no-plots", action="store_true", help="do not write plot SVGs")
    ap.add_argument("--plots-dir", type=str, default=str(LOGS_DIR / "wrs_plots"), help="directory to write SVG plots into")
    ap.add_argument("--time-bucket-sec", type=int, default=1, help="time-series bucket size in seconds (default: 1)")
    ap.add_argument(
        "--only-timeseries-compare-dashboard",
        action="store_true",
        help="only keep wrs_timeseries_compare_dashboard.svg (delete other SVGs in plots-dir after generating it)",
    )
    ap.add_argument("--ignore-cache", action="store_true", default=True, help="disable cache (default)")
    ap.add_argument("--use-cache", action="store_true", help="enable cache (sets IGNORE_CACHE=false)")
    args = ap.parse_args()

    ignore_cache = args.ignore_cache and not args.use_cache

    # Weights: either explicit overrides, or read from test3_simple.sh
    weights_from_args = not any(math.isnan(x) for x in [args.availability_weight, args.latency_weight, args.sync_weight, args.stake_weight])
    if weights_from_args:
        weights = Weights(
            availability=args.availability_weight,
            latency=args.latency_weight,
            sync=args.sync_weight,
            stake=args.stake_weight,
        )
    else:
        test3_contents = read_text(TEST3_SH)
        weights = parse_weights_from_test3_sh(test3_contents)

    # Providers for theory: either explicit --provider-json (and optional --provider-name), or parse from test3_simple.sh
    providers: List[Tuple[str, dict]] = []
    stake_values: List[int] | None = None
    if args.provider_json:
        default_names = ["provider-2220-tendermintrpc", "provider-2221-tendermintrpc", "provider-2222-tendermintrpc"]
        names = args.provider_name if args.provider_name else default_names[: len(args.provider_json)]
        if len(names) != len(args.provider_json):
            die(f"--provider-name count ({len(names)}) must match --provider-json count ({len(args.provider_json)})")
        if args.provider_stake:
            if len(args.provider_stake) != len(args.provider_json):
                die(f"--provider-stake count ({len(args.provider_stake)}) must match --provider-json count ({len(args.provider_json)})")
            stake_values = [int(x) for x in args.provider_stake]
        for name, p in zip(names, args.provider_json):
            providers.append((name, json.loads(read_text(Path(p)))))
    else:
        test3_contents = read_text(TEST3_SH)
        provider_pairs = parse_provider_test_response_files(test3_contents)
        for provider_cfg_path, responses_path in provider_pairs:
            provider_name = parse_provider_name_from_provider_cfg(read_text(provider_cfg_path))
            providers.append((provider_name, json.loads(read_text(responses_path))))

    # average_block_time comes from the lava spec used in SPECS_DIR in the script
    lava_spec_path = REPO_ROOT / "specs" / "testnet-2" / "specs" / "lava.json"
    avg_block_ms = parse_average_block_time_ms_from_lava_spec(lava_spec_path, chain_index="LAV1")

    if not args.no_run:
        run_test3(args.warmup, args.measured, args.parallel, ignore_cache=ignore_cache)

    consumer_log_path = Path(args.consumer_log) if args.consumer_log else CONSUMER_LOG
    provider_hits_path = Path(args.provider_hits) if args.provider_hits else PROVIDER_HITS

    observed = parse_observed_from_provider_hits(provider_hits_path)
    observed_health_log = parse_observed_from_consumer_log_health(consumer_log_path)

    theory, expected_probs = compute_theory(
        providers,
        weights,
        avg_block_ms,
        stake_values=stake_values,
        strategy=args.strategy,
        min_selection_chance=max(0.0, float(args.min_selection_chance)),
    )

    # Real score summaries from logs (average over measured window)
    consumer_text = read_text(consumer_log_path) if consumer_log_path.exists() else ""
    try:
        wstart, wend = find_measured_time_window_from_consumer_log(consumer_text, args.warmup, args.measured)
        breakdowns = parse_real_score_breakdowns(consumer_text, wstart, wend, slack_seconds=2)
        exclude_non_external = args.exclude_non_external_relays and not args.include_non_external_relays
        if exclude_non_external:
            events = parse_external_health_events(consumer_text)
            # measured-only events (exclude warmup)
            measured_events = events[args.warmup : args.warmup + args.measured]
            breakdowns = match_external_events_to_breakdowns(measured_events, breakdowns, max_skew_seconds=2.0)
        real = summarize_real_scores(breakdowns, observed)
    except Exception as e:
        print(f"\nWARNING: could not compute real score summaries from consumer log: {e}")
        real = {}

    print_report(args.measured, weights, avg_block_ms, theory, expected_probs, observed, observed_health_log, real_summary=real)

    # Write a markdown comparison table next to the SVG output.
    try:
        plots_dir = Path(args.plots_dir)
        write_theory_vs_real_table_md(plots_dir / "theory_vs_real.md", theory, expected_probs, real)
    except Exception as e:
        print(f"WARNING: could not write theory_vs_real.md: {e}")

    if not args.no_plots:
        plots_dir = Path(args.plots_dir)
        write_plots(plots_dir, theory, expected_probs, real, observed)
        # Time-series plots (scores/composite + distribution), 1s buckets by default.
        try:
            wstart, wend = find_measured_time_window_from_consumer_log(consumer_text, args.warmup, args.measured)
            samples = parse_real_score_breakdowns(consumer_text, wstart, wend, slack_seconds=2)
            exclude_non_external = args.exclude_non_external_relays and not args.include_non_external_relays
            if exclude_non_external:
                events = parse_external_health_events(consumer_text)
                measured_events = events[args.warmup : args.warmup + args.measured]
                samples = match_external_events_to_breakdowns(measured_events, samples, max_skew_seconds=2.0)
            providers_sorted = sorted({t.name for t in theory})
            write_timeseries_plots(
                plots_dir,
                consumer_text,
                wstart,
                wend,
                bucket_sec=max(1, int(args.time_bucket_sec)),
                providers=providers_sorted,
                samples=samples,
                theory=theory,
                expected_probs=expected_probs,
            )
        except Exception as e:
            print(f"WARNING: could not write time-series plots: {e}")

        if args.only_timeseries_compare_dashboard:
            # Leave only the comparison dashboard SVG in plots_dir.
            keep = "wrs_timeseries_compare_dashboard.svg"
            try:
                for p in plots_dir.glob("*.svg"):
                    if p.name != keep:
                        p.unlink(missing_ok=True)
            except Exception as e:
                print(f"WARNING: failed cleaning extra SVGs: {e}")
            print("\n=== Plots written ===")
            print(f"- {plots_dir / keep}")
        else:
            print("\n=== Plots written ===")
            print(f"- {plots_dir / 'theoretical_scores.svg'}")
            print(f"- {plots_dir / 'real_scores_from_logs.svg'}")
            print(f"- {plots_dir / 'distribution_expected_vs_observed.svg'}")
            print(f"- {plots_dir / 'wrs_dashboard.svg'}")
            print(f"- {plots_dir / 'wrs_timeseries_dashboard.svg'}")
            print(f"- {plots_dir / 'wrs_timeseries_compare_dashboard.svg'}")


if __name__ == "__main__":
    main()

