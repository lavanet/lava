#!/bin/bash
# scripts/test/chaos/tier_a_poisoning/run_all.sh
#
# Orchestrator for Tier A poisoning scenarios. Globs all `*.sh` siblings in
# this directory (excluding itself), runs each in sequence, captures exit code
# + wall time + stdout, and prints a summary table.
#
# Output: scenario stdout/stderr is redirected to per-scenario log files at
# $LOGS_DIR/scenario_<name>_<timestamp>.log (§8.3 of the infrastructure doc).
# The summary table is printed to stdout.
#
# Phase 1 scope: just smoke.sh. Phase 2 will add the 5 actual Tier A scenarios.
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.4

# pipefail: if anything piped (e.g. `scenario | tee`) fails, the pipeline rc
# reflects it. Deliberately NO `set -e` — we WANT to continue running remaining
# scenarios after one fails so the summary table is complete. Aggregate failure
# tracked via $OVERALL_RC instead.
set -o pipefail

__dir="$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source "$__dir"/../lib/common.sh

# Aggregate exit-code propagation. Stays 0 if every scenario passes; becomes
# the rc of the FIRST failing scenario otherwise. CI / parent scripts read
# this to decide pass/fail at the suite level.
OVERALL_RC=0

# Collect scenarios: every *.sh in this directory except run_all.sh itself.
SCENARIOS=()
for f in "$__dir"/*.sh; do
    [ "$(basename "$f")" = "run_all.sh" ] && continue
    SCENARIOS+=("$f")
done

if [ ${#SCENARIOS[@]} -eq 0 ]; then
    echo "[run_all.sh] no scenarios found in $__dir" >&2
    exit 1
fi

# Per-scenario result tracking. Parallel arrays keyed by scenario name.
declare -a NAMES
declare -a RESULTS    # PASS | FAIL
declare -a DURATIONS  # seconds (integer)
declare -a LOG_FILES

run_ts=$(date +"%Y%m%d_%H%M%S")
echo "[run_all.sh] running ${#SCENARIOS[@]} scenario(s) — logs under $LOGS_DIR/scenario_*_${run_ts}.log"

for scenario in "${SCENARIOS[@]}"; do
    name=$(basename "$scenario" .sh)
    log_file="$LOGS_DIR/scenario_${name}_${run_ts}.log"
    NAMES+=("$name")
    LOG_FILES+=("$log_file")

    echo "[run_all.sh] → running $name (log: $log_file)"
    start=$(date +%s)
    if "$scenario" >"$log_file" 2>&1; then
        rc=0
    else
        rc=$?
    fi
    end=$(date +%s)
    duration=$((end - start))
    DURATIONS+=("$duration")

    if [ "$rc" -eq 0 ]; then
        RESULTS+=("PASS")
        echo "[run_all.sh]   PASS in ${duration}s"
    else
        RESULTS+=("FAIL")
        echo "[run_all.sh]   FAIL (rc=$rc) in ${duration}s — see $log_file"
        # Capture the FIRST failing rc; subsequent failures don't overwrite so
        # operators can correlate the orchestrator's exit code with the first
        # offending scenario's exit code.
        [ "$OVERALL_RC" -eq 0 ] && OVERALL_RC=$rc
    fi
done

# Print the summary table.
echo
echo "==========================================================================="
echo "Tier A Poisoning Scenarios — Summary"
echo "==========================================================================="
printf "%-30s %-8s %-10s %s\n" "Scenario" "Result" "Duration" "Log"
printf "%-30s %-8s %-10s %s\n" "------------------------------" "--------" "----------" "----------------------------------------"
for i in "${!NAMES[@]}"; do
    printf "%-30s %-8s %-10s %s\n" \
        "${NAMES[$i]}" "${RESULTS[$i]}" "${DURATIONS[$i]}s" "$(basename "${LOG_FILES[$i]}")"
done
echo "==========================================================================="

if [ "$OVERALL_RC" -eq 0 ]; then
    echo "All scenarios passed."
else
    echo "One or more scenarios failed (overall rc=$OVERALL_RC)."
fi
exit "$OVERALL_RC"
