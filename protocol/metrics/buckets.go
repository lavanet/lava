package metrics

// LatencyBuckets is the shared histogram bucket set for all latency metrics
// in milliseconds. Starting at 1ms ensures cache hits (0–5ms) and fast
// in-process operations are resolved into distinct buckets rather than being
// collapsed into a single catch-all ≤10ms bucket.
var LatencyBuckets = []float64{1, 2, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000}
