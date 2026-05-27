# scripts/test/chaos/lib/metrics.sh
#
# Prometheus /metrics scraping helpers for chaos scenarios. One reader function
# (metric_get) — assertion helpers live in lib/assertions.sh.
#
# Source via:
#     source "$LIB_DIR"/common.sh
#     source "$LIB_DIR"/metrics.sh
# (depends on METRICS_PORT_C{1,2} from common.sh.)
#
# See: protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md §4.2.3

# _metrics_port_for_consumer — internal: <consumer-id> ∈ {1,2,c1,c2} → port.
_metrics_port_for_consumer() {
    case "$1" in
        1|c1) echo "$METRICS_PORT_C1" ;;
        2|c2) echo "$METRICS_PORT_C2" ;;
        *)    log "_metrics_port_for_consumer: invalid consumer-id '$1' (expected 1, 2, c1, c2)"; return 1 ;;
    esac
}

# metric_get <consumer-id> <metric-name> [<labels>]
#
# Reads the named metric from the consumer's /metrics endpoint and prints the
# numeric value to stdout. If the metric has no series (Prometheus counters
# don't emit until first increment) the function prints "0" — this matches the
# plan's contract that "absence == zero count" for counter assertions.
#
# Arguments:
#   <consumer-id>  — 1, 2, c1, or c2 (resolves to METRICS_PORT_C{1,2})
#   <metric-name>  — exact metric name; anchored at line start so partial
#                    metric-name overlap (e.g. lava_latest_block vs
#                    lava_latest_block_age) doesn't false-match.
#   <labels>       — optional. Substring matched against the labels block.
#                    Pass like '{spec="LAV1",apiInterface="rest"}' to filter
#                    to a single series, or empty/omit to aggregate ALL series
#                    of the metric (sum across labels — Prometheus convention).
#
# Multi-series metrics: when no labels are given (or when the labels filter
# matches multiple series), the function SUMS the values. Counters / gauges
# both work this way; sum-of-counters is monotonic, sum-of-gauges-by-spec
# gives a clean total.
#
# Output: a number on stdout, no trailing newline issues. Returns 0 on success;
# returns 1 + log line on consumer-id resolution failure or curl failure.
metric_get() {
    local consumer_id="$1"
    local metric_name="$2"
    local labels="${3:-}"
    if [ -z "$consumer_id" ] || [ -z "$metric_name" ]; then
        log "metric_get: usage: metric_get <consumer-id> <metric-name> [<labels>]"
        return 1
    fi
    local port
    port=$(_metrics_port_for_consumer "$consumer_id") || return 1

    local raw
    raw=$(curl -sf --connect-timeout 2 "http://127.0.0.1:${port}/metrics" 2>/dev/null)
    if [ -z "$raw" ]; then
        log "metric_get: /metrics at :${port} returned empty / unreachable"
        return 1
    fi

    # Filter to value lines starting with the metric name. Use a tab/space
    # boundary OR `{` boundary to avoid partial-name overlap.
    local series
    series=$(echo "$raw" | grep -E "^${metric_name}(\\{|[[:space:]])")
    if [ -z "$series" ]; then
        echo "0"
        return 0
    fi
    if [ -n "$labels" ]; then
        series=$(echo "$series" | grep -F "$labels")
        if [ -z "$series" ]; then
            echo "0"
            return 0
        fi
    fi

    # Sum the trailing numeric column across matching series. awk handles 0
    # matches → 0, 1 match → that value, N matches → sum.
    echo "$series" | awk '{ sum += $NF } END { printf "%g\n", (sum + 0) }'
}
