#!/usr/bin/env python3
"""
Shared WRS test analyzer/runner.

Runs a test's `run.sh` and then invokes `scripts/test3_wrs_analyze.py` with explicit
paths/weights/config JSONs so each test is fully self-contained.
"""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
from pathlib import Path
import re


REPO_ROOT = Path(__file__).resolve().parents[2]
BASE_ANALYZER = REPO_ROOT / "wrs_tests" / "_framework" / "wrs_analyze.py"


def die(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(2)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--test-dir", required=True, help="path to a wrs_tests/test_* directory")
    ap.add_argument("--warmup", type=int, default=100)
    ap.add_argument("--measured", type=int, default=5000)
    ap.add_argument("--parallel", type=int, default=100)
    ap.add_argument("--time-bucket-sec", type=int, default=3)
    ap.add_argument("--availability-weight", type=float, default=0.3)
    ap.add_argument("--latency-weight", type=float, default=0.3)
    ap.add_argument("--sync-weight", type=float, default=0.2)
    ap.add_argument("--stake-weight", type=float, default=0.2)
    ap.add_argument("--no-run", action="store_true")
    args = ap.parse_args()

    test_dir = Path(args.test_dir).resolve()
    run_sh = test_dir / "run.sh"
    outputs_dir = test_dir / "outputs"
    logs_dir = outputs_dir / "logs"

    if not BASE_ANALYZER.exists():
        die(f"missing base analyzer: {BASE_ANALYZER}")
    if not test_dir.exists():
        die(f"missing test-dir: {test_dir}")

    # Run the test (produces logs/hits into outputs/logs)
    if not args.no_run:
        if not run_sh.exists():
            die(f"missing {run_sh}")
        env = dict(os.environ)
        env["WARMUP_REQUESTS"] = str(args.warmup)
        env["NUM_REQUESTS"] = str(args.measured)
        env["PARALLELISM"] = str(args.parallel)
        env["AVAIL_WEIGHT"] = str(args.availability_weight)
        env["LAT_WEIGHT"] = str(args.latency_weight)
        env["SYNC_WEIGHT"] = str(args.sync_weight)
        env["STAKE_WEIGHT"] = str(args.stake_weight)
        subprocess.run(["bash", str(run_sh)], cwd=str(test_dir), env=env, check=True)

    plots_dir = outputs_dir / "wrs_plots"
    plots_dir.mkdir(parents=True, exist_ok=True)

    # Delegate analysis/plotting to the base analyzer against this test's artifacts
    provider_jsons = [
        str((test_dir / "configs" / "provider1.json").resolve()),
        str((test_dir / "configs" / "provider2.json").resolve()),
        str((test_dir / "configs" / "provider3.json").resolve()),
    ]
    for p in provider_jsons:
        if not Path(p).exists():
            die(f"missing provider json: {p}")

    cmd = [
        "python3",
        str(BASE_ANALYZER),
        "--no-run",
        "--only-timeseries-compare-dashboard",
        "--warmup",
        str(args.warmup),
        "--measured",
        str(args.measured),
        "--time-bucket-sec",
        str(args.time_bucket_sec),
        "--plots-dir",
        str(plots_dir),
        "--consumer-log",
        str((logs_dir / "CONSUMER.test3_simple.log").resolve()),
        "--provider-hits",
        str((logs_dir / "provider_hits.test3_simple.txt").resolve()),
        "--availability-weight",
        str(args.availability_weight),
        "--latency-weight",
        str(args.latency_weight),
        "--sync-weight",
        str(args.sync_weight),
        "--stake-weight",
        str(args.stake_weight),
    ]
    for pj in provider_jsons:
        cmd += ["--provider-json", pj]

    # Optional: pass explicit stake values for theoretical stake normalization (stake impact test).
    consumer_cfg = test_dir / "configs" / "consumer.yml"
    if consumer_cfg.exists():
        text = consumer_cfg.read_text(errors="replace")
        provider_names = ["provider-2220-tendermintrpc", "provider-2221-tendermintrpc", "provider-2222-tendermintrpc"]
        stakes: list[int] = []
        all_present = True
        for name in provider_names:
            # capture the block for this provider entry
            m = re.search(rf"(?ms)-\s*name:\s*{re.escape(name)}\s*$([\s\S]*?)(?=^\s*-\s*name:\s*|\Z)", text)
            if not m:
                all_present = False
                break
            m2 = re.search(r"(?m)^\s*stake:\s*([0-9]+)\s*$", m.group(1))
            if not m2:
                all_present = False
                break
            stakes.append(int(m2.group(1)))
        if all_present and stakes:
            for s in stakes:
                cmd += ["--provider-stake", str(s)]

    subprocess.run(cmd, cwd=str(REPO_ROOT), check=True)

    print("\nOutputs:")
    print(f"- logs: {logs_dir}")
    print(f"- plots: {plots_dir}")


if __name__ == "__main__":
    main()

