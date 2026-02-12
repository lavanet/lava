#!/usr/bin/env python3
"""
WRS test analyzer (per-test, parameterized).

Reads inputs from a test directory:
  - configs/provider{1,2,3}.json (test-mode configs)
  - outputs/logs/CONSUMER.test3_simple.log
  - outputs/logs/provider_hits.test3_simple.txt

Computes:
  - theoretical steady-state scores + expected distribution based on the JSON configs and weights
  - observed distribution from response headers
  - plots (including dashboards) into outputs/wrs_plots
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import math
import re
import subprocess
import sys
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Tuple


REPO_ROOT = Path(__file__).resolve().parents[2]
LAVA_SPEC = REPO_ROOT / "specs" / "testnet-2" / "specs" / "lava.json"


def die(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


def read_text(p: Path) -> str:
    return p.read_text(encoding="utf-8", errors="replace")


def clamp(x: float, lo: float, hi: float) -> float:
    return lo if x < lo else hi if x > hi else x


@dataclass(frozen=True)
class Weights:
    availability: float
    latency: float
    sync: float
    stake: float


@dataclass(frozen=True)
class ProviderTheory:
    name: str
    gap_blocks: int
    availability: float
    delay_ms: int
    stake_ratio: float
    raw_latency_sec: float
    raw_sync_sec: float
    normalized_latency: float
    normalized_sync: float
    normalized_availability: float
    normalized_stake: float
    composite: float
    expected_prob: float


def normalize_availability(av: float) -> float:
    min_acceptable = 0.80
    if av < min_acceptable:
        return 0.0
    return clamp((av - min_acceptable) / (1.0 - min_acceptable), 0.0, 1.0)


def normalize_latency_fixed(latency_sec: float) -> float:
    max_latency = 30.0
    if latency_sec <= 0:
        return 1.0
    return max(0.0, 1.0 - (latency_sec / max_latency))


def normalize_sync_adaptive(sync_lag_sec: float, p10: float, p90: float) -> float:
    if not (p10 > 0 and p90 > 0 and p90 > p10):
        max_sync = 1200.0
        if sync_lag_sec <= 0:
            return 1.0
        return max(0.0, 1.0 - (sync_lag_sec / max_sync))
    clamped = clamp(sync_lag_sec, p10, p90)
    return clamp(1.0 - (clamped - p10) / (p90 - p10), 0.0, 1.0)


def normalize_stake_equal(n: int) -> Tuple[float, float]:
    if n <= 0:
        return 0.0, 0.0
    ratio = 1.0 / float(n)
    return ratio, math.sqrt(ratio)


def parse_average_block_time_ms(chain_index: str = "LAV1") -> int:
    data = json.loads(read_text(LAVA_SPEC))
    for s in data.get("proposal", {}).get("specs", []):
        if s.get("index") == chain_index and "average_block_time" in s:
            return int(s["average_block_time"])
    die(f"could not find average_block_time for {chain_index} in {LAVA_SPEC}")


def compute_theory(provider_cfgs: List[Tuple[str, dict]], weights: Weights, avg_block_time_ms: int) -> Tuple[List[ProviderTheory], Dict[str, float]]:
    avg_block_time_sec = avg_block_time_ms / 1000.0
    gaps = [int(cfg.get("gap_blocks", 0)) for _, cfg in provider_cfgs]
    raw_syncs = [g * avg_block_time_sec for g in gaps]
    p10 = clamp(min(raw_syncs), 0.1, 60.0)
    p90 = clamp(max(raw_syncs), 30.0, 1200.0)
    if p90 <= p10:
        p90 = max(p10 + 1.0, 30.0)

    stake_ratio, norm_stake = normalize_stake_equal(len(provider_cfgs))

    composites: Dict[str, float] = {}
    out: List[ProviderTheory] = []
    for (name, cfg), gap, raw_sync in zip(provider_cfgs, gaps, raw_syncs):
        health = (cfg.get("responses", {}) or {}).get("health", {}) or {}
        availability = float(health.get("availability", 1.0))
        delay_ms = int(health.get("delay_ms", 0))
        raw_latency_sec = delay_ms / 1000.0

        nav = normalize_availability(availability)
        nlat = normalize_latency_fixed(raw_latency_sec)
        nsync = normalize_sync_adaptive(raw_sync, p10, p90)

        comp = (
            weights.availability * nav
            + weights.latency * nlat
            + weights.sync * nsync
            + weights.stake * norm_stake
        )
        composites[name] = comp
        out.append(
            ProviderTheory(
                name=name,
                gap_blocks=gap,
                availability=availability,
                delay_ms=delay_ms,
                stake_ratio=stake_ratio,
                raw_latency_sec=raw_latency_sec,
                raw_sync_sec=raw_sync,
                normalized_latency=nlat,
                normalized_sync=nsync,
                normalized_availability=nav,
                normalized_stake=norm_stake,
                composite=comp,
                expected_prob=0.0,
            )
        )

    total = sum(composites.values())
    expected = {k: v / total for k, v in composites.items()}
    out = [ProviderTheory(**{**t.__dict__, "expected_prob": expected.get(t.name, 0.0)}) for t in out]
    return out, expected


def observed_from_hits(path: Path) -> Counter:
    lines = [ln.strip() for ln in read_text(path).splitlines() if ln.strip()]
    return Counter(lines)


def _svg_escape(s: str) -> str:
    return (
        s.replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace('"', "&quot;")
        .replace("'", "&#39;")
    )


def write_svg_grouped_bars(out_path: Path, title: str, providers: List[str], series: List[Tuple[str, List[float]]], y_max: float = 1.0) -> None:
    width = 1200
    height = 520
    margin = 60
    plot_w = width - 2 * margin
    plot_h = height - 2 * margin
    if not providers or not series:
        return
    nprov = len(providers)
    nser = len(series)
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
    lines.append(f'<line x1="{margin}" y1="{margin}" x2="{margin}" y2="{margin+plot_h}" stroke="#333" stroke-width="1"/>')
    lines.append(f'<line x1="{margin}" y1="{margin+plot_h}" x2="{margin+plot_w}" y2="{margin+plot_h}" stroke="#333" stroke-width="1"/>')

    for t in [0.0, 0.25, 0.5, 0.75, 1.0]:
        yy = y(t * y_max)
        lines.append(f'<line x1="{margin-5}" y1="{yy}" x2="{margin}" y2="{yy}" stroke="#333" stroke-width="1"/>')
        lines.append(f'<text x="{margin-10}" y="{yy+4}" font-family="Arial" font-size="12" text-anchor="end">{t:.2f}</text>')

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

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text("\n".join(lines), encoding="utf-8")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--test-dir", required=True)
    ap.add_argument("--warmup", type=int, default=100)
    ap.add_argument("--measured", type=int, default=5000)
    ap.add_argument("--parallel", type=int, default=100)
    ap.add_argument("--time-bucket-sec", type=int, default=3)
    ap.add_argument("--no-run", action="store_true")
    ap.add_argument("--availability-weight", type=float, default=0.3)
    ap.add_argument("--latency-weight", type=float, default=0.3)
    ap.add_argument("--sync-weight", type=float, default=0.2)
    ap.add_argument("--stake-weight", type=float, default=0.2)
    args = ap.parse_args()

    test_dir = Path(args.test_dir).resolve()
    cfg_dir = test_dir / "configs"
    out_dir = test_dir / "outputs"
    logs_dir = out_dir / "logs"
    plots_dir = out_dir / "wrs_plots"

    if not cfg_dir.exists():
        die(f"missing {cfg_dir}")

    if not args.no_run:
        env = dict(**dict())
        env.update(**dict(**__import__("os").environ))
        env["WARMUP_REQUESTS"] = str(args.warmup)
        env["NUM_REQUESTS"] = str(args.measured)
        env["PARALLELISM"] = str(args.parallel)
        env["AVAIL_WEIGHT"] = str(args.availability_weight)
        env["LAT_WEIGHT"] = str(args.latency_weight)
        env["SYNC_WEIGHT"] = str(args.sync_weight)
        env["STAKE_WEIGHT"] = str(args.stake_weight)
        subprocess.run(["bash", str(test_dir / "run.sh")], cwd=str(test_dir), env=env, check=True)

    weights = Weights(args.availability_weight, args.latency_weight, args.sync_weight, args.stake_weight)
    avg_block_ms = parse_average_block_time_ms()

    provider_cfgs: List[Tuple[str, dict]] = []
    for i in [1, 2, 3]:
        cfg = json.loads(read_text(cfg_dir / f"provider{i}.json"))
        # Hardcode names to match consumer static-providers list
        provider_cfgs.append((f"provider-222{i-1}-tendermintrpc", cfg))

    theory, expected = compute_theory(provider_cfgs, weights, avg_block_ms)
    hits = observed_from_hits(logs_dir / "provider_hits.test3_simple.txt")
    total = sum(hits.values()) or 1

    providers = [t.name for t in theory]
    tmap = {t.name: t for t in theory}

    write_svg_grouped_bars(
        plots_dir / "theoretical_scores.svg",
        "Theoretical normalized scores + composite",
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
    write_svg_grouped_bars(
        plots_dir / "distribution_expected_vs_observed.svg",
        "Expected vs observed selection probability",
        providers,
        [
            ("expected", [expected.get(p, 0.0) for p in providers]),
            ("observed", [hits.get(p, 0) / total for p in providers]),
        ],
        y_max=1.0,
    )

    print("plots:", plots_dir)


if __name__ == "__main__":
    main()

