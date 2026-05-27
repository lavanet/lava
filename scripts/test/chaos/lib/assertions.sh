# scripts/test/chaos/lib/assertions.sh
#
# Test-style assertions for chaos scenarios. Each function returns 0 on pass
# and exits the calling script with a non-zero exit code on fail (so a
# scenario script that does `set -e` doesn't need to trap each call).
#
# The exit-on-fail behavior is deliberate: scenarios are pass/fail end-to-end
# and the run_all.sh aggregator (subtask 4.1) reads exit codes to compute the
# summary table. Continuing past a failed assertion leaves the env in a
# poisoned state and confuses subsequent assertions.
#
# Source via:
#     source "$LIB_DIR"/common.sh
#     source "$LIB_DIR"/metrics.sh        # for metric_get
#     source "$LIB_DIR"/assertions.sh
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.2.3

# assert_metric_eq <consumer-id> <metric-name> <labels> <expected>
# Reads the metric via metric_get; FAILS the scenario if the value doesn't
# match <expected>. Pass empty string for <labels> if the metric is unlabeled.
# Comparison is string-based; for integer metrics this works since metric_get
# emits clean integer/float strings via awk's %g.
assert_metric_eq() {
    local consumer_id="$1"
    local metric_name="$2"
    local labels="$3"
    local expected="$4"
    if [ -z "$consumer_id" ] || [ -z "$metric_name" ] || [ -z "$expected" ]; then
        log "assert_metric_eq: usage: assert_metric_eq <consumer-id> <metric> <labels> <expected>"
        exit 2
    fi
    local actual
    actual=$(metric_get "$consumer_id" "$metric_name" "$labels") || exit 2
    if [ "$actual" != "$expected" ]; then
        log "ASSERT FAILED: ${metric_name}${labels} on consumer ${consumer_id}: expected '${expected}', got '${actual}'"
        exit 1
    fi
    return 0
}

# assert_metric_increased_since <consumer-id> <metric-name> <labels> <baseline>
# Reads the metric via metric_get; FAILS the scenario if the current value is
# not strictly greater than <baseline>. Used for counter-increment assertions
# across a test step — capture a baseline before the trigger, run the trigger,
# then assert the counter advanced.
#
# Numeric (integer) comparison via shell arithmetic. metric_get emits floats
# only for fractional values (e.g. gauges with rates) — counters are always
# integers, so this comparison is safe for the typical scenario use case.
# Floats fall through to bc for portable comparison.
assert_metric_increased_since() {
    local consumer_id="$1"
    local metric_name="$2"
    local labels="$3"
    local baseline="$4"
    if [ -z "$consumer_id" ] || [ -z "$metric_name" ] || [ -z "$baseline" ]; then
        log "assert_metric_increased_since: usage: assert_metric_increased_since <consumer-id> <metric> <labels> <baseline>"
        exit 2
    fi
    local actual
    actual=$(metric_get "$consumer_id" "$metric_name" "$labels") || exit 2
    # Pure integer fast path
    if [[ "$actual" =~ ^[0-9]+$ ]] && [[ "$baseline" =~ ^[0-9]+$ ]]; then
        if [ "$actual" -le "$baseline" ]; then
            log "ASSERT FAILED: ${metric_name}${labels} on consumer ${consumer_id}: did not increase past baseline=${baseline}, current=${actual}"
            exit 1
        fi
        return 0
    fi
    # Float path via bc (always available on macOS + Linux).
    local cmp
    cmp=$(echo "$actual > $baseline" | bc -l 2>/dev/null)
    if [ "$cmp" != "1" ]; then
        log "ASSERT FAILED: ${metric_name}${labels} on consumer ${consumer_id}: did not increase past baseline=${baseline}, current=${actual}"
        exit 1
    fi
    return 0
}

# assert_log_present <log-file> <pattern>
# Greps <log-file> for an extended-regex <pattern>. FAILS if no match. On fail,
# prints the last 20 lines of the log to stderr so operators have immediate
# context for debugging without re-running the test.
assert_log_present() {
    local log_file="$1"
    local pattern="$2"
    if [ -z "$log_file" ] || [ -z "$pattern" ]; then
        log "assert_log_present: usage: assert_log_present <log-file> <pattern>"
        exit 2
    fi
    if [ ! -f "$log_file" ]; then
        log "ASSERT FAILED: log file does not exist: ${log_file}"
        exit 1
    fi
    if ! grep -qE "$pattern" "$log_file"; then
        log "ASSERT FAILED: pattern not found in ${log_file}: ${pattern}"
        log "  last 20 lines for context:"
        tail -n 20 "$log_file" | sed 's/^/    /' >&2
        exit 1
    fi
    return 0
}
