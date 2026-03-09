# Weighted Selector Testing Plan

## PR #2112: Replace Tiers with Weighted Random Selection

**Document Version:** 1.0
**Date:** 2026-01-22
**Author:** Code Review

---

## Table of Contents

1. [Testing Objectives](#1-testing-objectives)
2. [Test Categories](#2-test-categories)
3. [Unit Tests](#3-unit-tests)
4. [Integration Tests](#4-integration-tests)
5. [Statistical Distribution Tests](#5-statistical-distribution-tests)
6. [Edge Case Tests](#6-edge-case-tests)
7. [Performance Tests](#7-performance-tests)
8. [Manual Validation Steps](#8-manual-validation-steps)
9. [Test Execution Checklist](#9-test-execution-checklist)
10. [Results Summary](#10-results-summary)

---

## 1. Testing Objectives

### Goals
- Verify weighted selection produces correct probability distributions
- Validate adaptive normalization (P10-P90) works correctly
- Ensure no regressions from tier-based system removal
- Confirm edge cases are handled safely
- Validate performance meets requirements

### Success Criteria
- All existing tests pass
- New weighted selector tests pass with correct distributions
- No panics or crashes under any input conditions
- Performance within acceptable bounds (< 1ms per selection)

---

## 2. Test Categories

| Category | Purpose | Files |
|----------|---------|-------|
| **Unit** | Individual component correctness | `weighted_selector_test.go`, `adaptive_max_calculator_test.go` |
| **Integration** | Component interaction | `provider_optimizer_test.go` |
| **Statistical** | Distribution validation | Custom distribution tests |
| **Edge Case** | Boundary condition handling | NaN, Inf, zero, overflow tests |
| **Performance** | Speed and memory | Benchmark tests |

---

## 3. Unit Tests

### 3.1 WeightedSelector Tests

#### Test: `TestNewWeightedSelector`
| Aspect | Expected | Actual | Status |
|--------|----------|--------|--------|
| Default weights sum | 1.0 | | ☐ |
| Availability weight | 0.3 | | ☐ |
| Latency weight | 0.3 | | ☐ |
| Sync weight | 0.2 | | ☐ |
| Stake weight | 0.2 | | ☐ |
| Min selection chance | 0.01 | | ☐ |

**Run Command:**
```bash
go test -v -run TestNewWeightedSelector ./protocol/provideroptimizer/...
```

---

#### Test: `TestWeightNormalization`
| Aspect | Expected | Actual | Status |
|--------|----------|--------|--------|
| Input weights (0.5, 0.5, 0.5, 0.5) | | | |
| Normalized total | 1.0 | | ☐ |
| Each normalized weight | 0.25 | | ☐ |

**Run Command:**
```bash
go test -v -run TestWeightNormalization ./protocol/provideroptimizer/...
```

---

#### Test: `TestCalculateScorePerfectProvider`
| Aspect | Expected | Actual | Status |
|--------|----------|--------|--------|
| Availability (1.0) normalized | 1.0 | | ☐ |
| Latency (0.0s) normalized | 1.0 | | ☐ |
| Sync (0.0s) normalized | 1.0 | | ☐ |
| Stake (10%) normalized | ~0.316 (sqrt) | | ☐ |
| Composite score | ~0.82 | | ☐ |

**Calculation Breakdown:**
```
availability: 1.0 * 0.3 = 0.30
latency:      1.0 * 0.3 = 0.30
sync:         1.0 * 0.2 = 0.20
stake:        sqrt(0.1) * 0.2 = 0.316 * 0.2 = 0.063
total: 0.30 + 0.30 + 0.20 + 0.063 = 0.863
```

**Run Command:**
```bash
go test -v -run TestCalculateScorePerfectProvider ./protocol/provideroptimizer/...
```

---

#### Test: `TestNormalizeLatency`
| Input (seconds) | Expected Score | Actual | Status |
|-----------------|----------------|--------|--------|
| 0.0 | 1.0 (perfect) | | ☐ |
| 3.0 | 0.9 | | ☐ |
| 15.0 | 0.5 | | ☐ |
| 30.0 | 0.0 (worst) | | ☐ |
| 60.0 | 0.0 (clamped) | | ☐ |

**Formula:** `normalized = 1.0 - (latency / 30.0)`

**Run Command:**
```bash
go test -v -run TestNormalizeLatency ./protocol/provideroptimizer/...
```

---

#### Test: `TestNormalizeSync`
| Input (seconds) | Expected Score | Actual | Status |
|-----------------|----------------|--------|--------|
| 0.0 | 1.0 (perfect) | | ☐ |
| 120.0 | 0.9 | | ☐ |
| 600.0 | 0.5 | | ☐ |
| 1200.0 | 0.0 (worst) | | ☐ |
| 2400.0 | 0.0 (clamped) | | ☐ |

**Formula:** `normalized = 1.0 - (sync / 1200.0)`

**Run Command:**
```bash
go test -v -run TestNormalizeSync ./protocol/provideroptimizer/...
```

---

#### Test: `TestNormalizeAvailability`
| Input | Expected Score | Actual | Status |
|-------|----------------|--------|--------|
| 1.00 (100%) | 1.0 | | ☐ |
| 0.95 (95%) | 0.75 | | ☐ |
| 0.90 (90%) | 0.5 | | ☐ |
| 0.80 (80%) | 0.0 (threshold) | | ☐ |
| 0.70 (70%) | 0.0 (below threshold) | | ☐ |

**Formula:** `normalized = (availability - 0.80) / 0.20`

**Run Command:**
```bash
go test -v -run TestNormalizeAvailability ./protocol/provideroptimizer/...
```

---

#### Test: `TestNormalizeStake` (Square Root Scaling)
| Stake Ratio | Linear Score | Sqrt Score | Boost Factor | Status |
|-------------|--------------|------------|--------------|--------|
| 1% (0.01) | 0.01 | 0.1 | 10x | ☐ |
| 10% (0.10) | 0.10 | 0.316 | 3.16x | ☐ |
| 25% (0.25) | 0.25 | 0.5 | 2x | ☐ |
| 50% (0.50) | 0.50 | 0.707 | 1.41x | ☐ |
| 80% (0.80) | 0.80 | 0.894 | 1.12x | ☐ |
| 100% (1.00) | 1.00 | 1.0 | 1x | ☐ |

**Formula:** `normalized = sqrt(stake / totalStake)`

**Purpose:** Reduces whale dominance by ~17% while maintaining stake incentives.

**Run Command:**
```bash
go test -v -run TestNormalizeStake ./protocol/provideroptimizer/...
```

---

### 3.2 AdaptiveMaxCalculator Tests

#### Test: `TestAdaptiveMaxCalculator_BasicFunctionality`
| Aspect | Expected | Actual | Status |
|--------|----------|--------|--------|
| Samples added | [0.5, 1.0, 1.5, 2.0, 2.5] | | ☐ |
| P95 percentile | ~2.5 ± 0.5 | | ☐ |
| Result >= minMax | true | | ☐ |
| Result <= maxMax | true | | ☐ |

**Run Command:**
```bash
go test -v -run TestAdaptiveMaxCalculator_BasicFunctionality ./utils/score/...
```

---

#### Test: `TestAdaptiveMaxCalculator_ExponentialDecay`
| Time Elapsed | Decay Factor | Description | Status |
|--------------|--------------|-------------|--------|
| 0 hours | 1.0 | No decay | ☐ |
| 1 hour (1 half-life) | 0.5 | 50% weight | ☐ |
| 2 hours (2 half-lives) | 0.25 | 25% weight | ☐ |
| 5 hours (5 half-lives) | 0.03125 | ~3% weight | ☐ |

**Formula:** `decayFactor = exp(-(ln(2) * timeDiff) / halfLife)`

**Run Command:**
```bash
go test -v -run TestAdaptiveMaxCalculator_ExponentialDecay ./utils/score/...
```

---

#### Test: `TestAdaptiveMaxCalculator_GetAdaptiveBounds`
| Provider Distribution | Expected P10 | Expected P90 | Actual P10 | Actual P90 | Status |
|----------------------|--------------|--------------|------------|------------|--------|
| A: 0.3s (300 samples) | | | | | |
| B: 1.0s (300 samples) | | | | | |
| C: 2.0s (300 samples) | | | | | |
| **Combined** | ~0.3 ± 0.2 | ~2.1 ± 0.5 | | | ☐ |

**Run Command:**
```bash
go test -v -run TestAdaptiveMaxCalculator_GetAdaptiveBounds ./utils/score/...
```

---

## 4. Integration Tests

### 4.1 Provider Optimizer Integration

#### Test: `TestProviderOptimizerBasicProbeData`
**Scenario:** 10 providers, damage providers 5-7 with bad latency, improve 0-2 with good latency

| Provider Group | Latency | Expected Selection Rate | Actual | Status |
|----------------|---------|------------------------|--------|--------|
| 0-2 (good) | 5ms | > 25% combined | | ☐ |
| 3-4 (default) | 10ms | ~20% combined | | ☐ |
| 5-7 (bad) | 30ms | < 25% combined | | ☐ |
| 8-9 (default) | 10ms | ~20% combined | | ☐ |

**Run Command:**
```bash
go test -v -run TestProviderOptimizerBasicProbeData ./protocol/provideroptimizer/...
```

---

#### Test: `TestProviderOptimizerBasicRelayData`
**Scenario:** 10 providers with extreme latency differences (1ms vs 1000ms)

| Provider Group | Latency | Expected Result | Actual | Status |
|----------------|---------|-----------------|--------|--------|
| 0-2 (excellent) | 1ms | > 30% selections | | ☐ |
| 5-7 (terrible) | 1000ms | < good providers | | ☐ |
| Good > Bad | - | true | | ☐ |

**Run Command:**
```bash
go test -v -run TestProviderOptimizerBasicRelayData ./protocol/provideroptimizer/...
```

---

### 4.2 Full Selection Pipeline Test

#### Test: Relay → T-Digest → Normalization → Selection
```
Step 1: AppendRelayData(provider, latency, cu, syncBlock)
  └─→ Updates provider's ScoreStore
  └─→ Feeds sample to globalLatencyCalculator (T-Digest)

Step 2: ChooseProvider(providers, ignored, cu, requestedBlock)
  └─→ GetReputationReportForProvider() → QoS metrics
  └─→ getAdaptiveLatencyBounds() → P10, P90 from T-Digest
  └─→ normalizeLatency() → Uses P10-P90 for normalization
  └─→ CalculateScore() → Composite score
  └─→ SelectProvider() → Weighted random selection
```

| Step | Component | Expected Behavior | Actual | Status |
|------|-----------|-------------------|--------|--------|
| 1 | ScoreStore | Updates with decaying average | | ☐ |
| 2 | T-Digest | Receives adjusted sample | | ☐ |
| 3 | P10-P90 | Returns valid bounds (P90 > P10) | | ☐ |
| 4 | Normalize | Returns value in [0, 1] | | ☐ |
| 5 | Composite | Score in [minChance, 1.0] | | ☐ |
| 6 | Selection | Provider selected based on weight | | ☐ |

---

## 5. Statistical Distribution Tests

### 5.1 Selection Distribution Validation

#### Test: `TestSelectProviderDistribution`
**Setup:** 3 providers with scores 0.8, 0.4, 0.2 (total = 1.4)

| Provider | Score | Expected % | Tolerance | Actual % | Status |
|----------|-------|------------|-----------|----------|--------|
| high_score | 0.8 | 57.1% (0.8/1.4) | ± 5% | | ☐ |
| medium_score | 0.4 | 28.6% (0.4/1.4) | ± 5% | | ☐ |
| low_score | 0.2 | 14.3% (0.2/1.4) | ± 5% | | ☐ |

**Iterations:** 10,000
**Seed:** 1234567 (deterministic)

**Run Command:**
```bash
go test -v -run TestSelectProviderDistribution ./protocol/provideroptimizer/...
```

---

#### Test: `TestSelectProviderEqualScores`
**Setup:** 3 providers with equal scores (0.5 each)

| Provider | Score | Expected % | Tolerance | Actual % | Status |
|----------|-------|------------|-----------|----------|--------|
| provider1 | 0.5 | 33.3% | ± 5% | | ☐ |
| provider2 | 0.5 | 33.3% | ± 5% | | ☐ |
| provider3 | 0.5 | 33.3% | ± 5% | | ☐ |

**Iterations:** 3,000

**Run Command:**
```bash
go test -v -run TestSelectProviderEqualScores ./protocol/provideroptimizer/...
```

---

#### Test: `TestMinSelectionChanceIsAWeightFloorNotAProbabilityGuarantee`
**Setup:** 2 providers - dominant (1.0) and minimum floor (0.01)

| Metric | Expected | Actual | Status |
|--------|----------|--------|--------|
| Min floor effective probability | 0.01/(1.0+0.01) = 0.99% | | ☐ |
| Is less than minSelectionChance | true (< 1%) | | ☐ |
| Statistical tolerance | ± 0.2% | | ☐ |

**Iterations:** 100,000

**Run Command:**
```bash
go test -v -run TestMinSelectionChanceIsAWeightFloorNotAProbabilityGuarantee ./protocol/provideroptimizer/...
```

---

### 5.2 Chi-Square Goodness of Fit Test

**Purpose:** Validate selection follows expected probability distribution

```go
// Manual validation procedure
func validateDistribution(observed map[string]int, expected map[string]float64, n int) {
    chiSquare := 0.0
    for provider, obs := range observed {
        exp := expected[provider] * float64(n)
        chiSquare += math.Pow(float64(obs)-exp, 2) / exp
    }
    // degrees of freedom = num_providers - 1
    // For 3 providers at α=0.05, critical value = 5.991
    // If chiSquare < 5.991, distribution is valid
}
```

| Test | Chi-Square | Critical Value (α=0.05) | Pass? | Status |
|------|------------|------------------------|-------|--------|
| 3 providers unequal | | 5.991 | | ☐ |
| 3 providers equal | | 5.991 | | ☐ |
| 10 providers | | 16.919 | | ☐ |

---

## 6. Edge Case Tests

### 6.1 Input Validation

#### Test: Invalid Weight Configurations
| Config | Expected Behavior | Actual | Status |
|--------|-------------------|--------|--------|
| All zeros (0,0,0,0) | Fallback to defaults | | ☐ |
| Negative weight (-0.1,0.6,0.3,0.2) | Fallback to defaults | | ☐ |
| NaN weight | Fallback to defaults | | ☐ |
| +Inf weight | Fallback to defaults | | ☐ |
| -Inf weight | Fallback to defaults | | ☐ |

**Run Commands:**
```bash
go test -v -run TestNewWeightedSelectorZeroTotalWeightFallsBackToDefaultWeightsButKeepsOtherConfig ./protocol/provideroptimizer/...
go test -v -run TestNewWeightedSelectorNegativeWeightFallsBackToDefaultWeightsButKeepsOtherConfig ./protocol/provideroptimizer/...
go test -v -run TestNewWeightedSelectorNaNWeightFallsBackToDefaultWeightsButKeepsOtherConfig ./protocol/provideroptimizer/...
go test -v -run TestNewWeightedSelectorInfWeightFallsBackToDefaultWeightsButKeepsOtherConfig ./protocol/provideroptimizer/...
```

---

#### Test: Provider List Edge Cases
| Scenario | Expected Behavior | Actual | Status |
|----------|-------------------|--------|--------|
| Empty provider list | Return "" | | ☐ |
| Single provider | Return that provider | | ☐ |
| All zero scores | Uniform random selection | | ☐ |
| All ignored providers | Return "" | | ☐ |

**Run Commands:**
```bash
go test -v -run TestSelectProviderEmptyList ./protocol/provideroptimizer/...
go test -v -run TestSelectProviderSingleProvider ./protocol/provideroptimizer/...
go test -v -run TestSelectProviderZeroScores ./protocol/provideroptimizer/...
```

---

#### Test: Stake Overflow Handling
| Scenario | Expected | Actual | Status |
|----------|----------|--------|--------|
| MaxInt64 stake | No panic, score = 1.0 | | ☐ |
| No NaN produced | true | | ☐ |
| No Inf produced | true | | ☐ |

**Run Command:**
```bash
go test -v -run TestNormalizeStakeMaxInt64DoesNotOverflowOrNaN ./protocol/provideroptimizer/...
```

---

### 6.2 Adaptive Calculator Edge Cases

| Scenario | Expected | Actual | Status |
|----------|----------|--------|--------|
| Nil calculator - AddSample | No error, returns nil | | ☐ |
| Nil calculator - GetAdaptiveMax | Returns 0.0 | | ☐ |
| Nil calculator - GetAdaptiveBounds | Returns (0.5, 3.0) safe defaults | | ☐ |
| Negative sample | Returns error | | ☐ |
| Old sample (before last update) | Handled gracefully, no decay | | ☐ |
| P90 ≤ P10 | Adjusted to P90 = P10 + 1.0 | | ☐ |

**Run Commands:**
```bash
go test -v -run TestAdaptiveMaxCalculator_NilHandling ./utils/score/...
go test -v -run TestAdaptiveMaxCalculator_NegativeSample ./utils/score/...
go test -v -run TestAdaptiveMaxCalculator_OldSampleHandling ./utils/score/...
```

---

## 7. Performance Tests

### 7.1 Benchmark Results

| Benchmark | Target | Actual | Status |
|-----------|--------|--------|--------|
| CalculateScore (single) | < 1µs | | ☐ |
| SelectProvider (50 providers) | < 10µs | | ☐ |
| CalculateProviderScores (50 providers) | < 100µs | | ☐ |

**Run Commands:**
```bash
go test -bench=BenchmarkCalculateScore -benchmem ./protocol/provideroptimizer/...
go test -bench=BenchmarkSelectProvider -benchmem ./protocol/provideroptimizer/...
go test -bench=BenchmarkCalculateProviderScores -benchmem ./protocol/provideroptimizer/...
```

**Expected Output Format:**
```
BenchmarkCalculateScore-8         5000000    xxx ns/op    xxx B/op    xxx allocs/op
BenchmarkSelectProvider-8         1000000    xxx ns/op    xxx B/op    xxx allocs/op
BenchmarkCalculateProviderScores-8  100000   xxx ns/op    xxx B/op    xxx allocs/op
```

---

### 7.2 Concurrency Safety

#### Test: `TestSelectProviderConcurrentDoesNotPanicAndReturnsValidProvider`
| Metric | Expected | Actual | Status |
|--------|----------|--------|--------|
| Goroutines | 32 | | ☐ |
| Iterations per goroutine | 2000 | | ☐ |
| Total selections | 64,000 | | ☐ |
| Invalid selections | 0 | | ☐ |
| Panics | 0 | | ☐ |

**Run Command:**
```bash
go test -v -run TestSelectProviderConcurrentDoesNotPanicAndReturnsValidProvider ./protocol/provideroptimizer/...
```

---

### 7.3 Memory Profile

| Component | Expected Memory | Actual | Status |
|-----------|-----------------|--------|--------|
| T-Digest (compression=100) | ~20KB | | ☐ |
| ProviderStakeCache (100 providers) | ~8KB | | ☐ |
| WeightedSelector | < 1KB | | ☐ |

**Run Command:**
```bash
go test -bench=. -benchmem -memprofile=mem.out ./protocol/provideroptimizer/...
go tool pprof mem.out
```

---

## 8. Manual Validation Steps

### 8.1 Visual Distribution Verification

**Step 1:** Run distribution test with verbose output
```bash
go test -v -run TestSelectProviderDistribution ./protocol/provideroptimizer/... 2>&1 | grep -E "(high_score|medium_score|low_score)"
```

**Expected Output:**
```
Selection distribution over 10000 iterations:
  high_score: ~57.XX% (expected ~57.1%)
  medium_score: ~28.XX% (expected ~28.6%)
  low_score: ~14.XX% (expected ~14.3%)
```

**Step 2:** Verify percentages are within tolerance (± 5%)

| Provider | Expected | Lower Bound | Upper Bound | Actual | Pass? |
|----------|----------|-------------|-------------|--------|-------|
| high_score | 57.1% | 52.1% | 62.1% | | ☐ |
| medium_score | 28.6% | 23.6% | 33.6% | | ☐ |
| low_score | 14.3% | 9.3% | 19.3% | | ☐ |

---

### 8.2 Adaptive Normalization Verification

**Step 1:** Add samples and check P10-P90 bounds
```go
// Test code
calc := NewAdaptiveMaxCalculator(...)
for i := 0; i < 100; i++ {
    calc.AddSample(float64(i)/10.0, time.Now()) // 0.0 to 10.0
}
p10, p90 := calc.GetAdaptiveBounds()
fmt.Printf("P10: %.3f, P90: %.3f\n", p10, p90)
```

**Expected:** P10 ≈ 1.0, P90 ≈ 9.0 (10th and 90th percentile of 0-10 range)

| Metric | Expected | Actual | Pass? |
|--------|----------|--------|-------|
| P10 | ~1.0 | | ☐ |
| P90 | ~9.0 | | ☐ |
| P90 > P10 | true | | ☐ |

---

### 8.3 Strategy Behavior Verification

| Strategy | Latency Adjustment | Sync Adjustment | Expected Effect |
|----------|-------------------|-----------------|-----------------|
| Balanced | None | None | Equal weighting | ☐ |
| Latency | Boost (power 0.8) | None | Favor low latency | ☐ |
| SyncFreshness | None | Boost (power 0.8) | Favor fresh sync | ☐ |
| Distributed | Flatten (power 1.2) | Flatten (power 1.2) | More diversity | ☐ |
| Privacy | None | None | Single provider only | ☐ |

**Run Command:**
```bash
go test -v -run "TestStrategy.*Adjustment" ./protocol/provideroptimizer/...
```

---

## 9. Test Execution Checklist

### Pre-Test Setup
- [ ] Ensure Go modules are up to date: `go mod tidy`
- [ ] Verify tdigest dependency is available
- [ ] Run `go build ./...` to check for compile errors

### Full Test Suite
```bash
# Run all tests in relevant packages
go test -v ./protocol/provideroptimizer/... ./utils/score/... 2>&1 | tee test_results.log

# Count results
echo "=== Test Summary ==="
grep -c "PASS:" test_results.log
grep -c "FAIL:" test_results.log
grep -c "SKIP:" test_results.log
```

### Expected Results
| Package | Tests | Expected Pass | Expected Fail | Expected Skip |
|---------|-------|---------------|---------------|---------------|
| provideroptimizer | ~45 | ~45 | 0 | 0 |
| utils/score | ~20 | ~20 | 0 | 0 |

---

## 10. Results Summary

### Execution Date: ________________

### Overall Status: ☐ PASS / ☐ FAIL

### Unit Tests
| Category | Total | Pass | Fail | Skip |
|----------|-------|------|------|------|
| WeightedSelector | | | | |
| AdaptiveMaxCalculator | | | | |
| ProviderStakeCache | | | | |

### Integration Tests
| Test | Status | Notes |
|------|--------|-------|
| Basic Probe Data | ☐ | |
| Basic Relay Data | ☐ | |
| Full Pipeline | ☐ | |

### Statistical Tests
| Test | Chi-Square | Pass? | Notes |
|------|------------|-------|-------|
| Distribution (unequal) | | ☐ | |
| Distribution (equal) | | ☐ | |
| Min selection floor | | ☐ | |

### Performance
| Benchmark | Result | Target Met? |
|-----------|--------|-------------|
| CalculateScore | | ☐ |
| SelectProvider | | ☐ |
| CalculateProviderScores | | ☐ |

### Edge Cases
| Category | All Pass? | Failed Tests |
|----------|-----------|--------------|
| Invalid weights | ☐ | |
| Empty/single provider | ☐ | |
| Overflow handling | ☐ | |
| Nil handling | ☐ | |

### Issues Found
| Issue # | Description | Severity | Resolution |
|---------|-------------|----------|------------|
| | | | |

---

## Appendix A: Test Commands Quick Reference

```bash
# All tests
go test -v ./protocol/provideroptimizer/... ./utils/score/...

# Weighted selector only
go test -v ./protocol/provideroptimizer/... -run "TestWeighted|TestNormalize|TestSelect|TestCalculate"

# Adaptive calculator only
go test -v ./utils/score/... -run "TestAdaptive"

# Benchmarks only
go test -bench=. ./protocol/provideroptimizer/... ./utils/score/...

# With coverage
go test -cover -coverprofile=coverage.out ./protocol/provideroptimizer/... ./utils/score/...
go tool cover -html=coverage.out

# Race detection
go test -race ./protocol/provideroptimizer/... ./utils/score/...
```

---

## Appendix B: Normalization Formulas Reference

### Availability (Phase 1 - Simple Rescaling)
```
input:  availability ∈ [0, 1]
output: score ∈ [0, 1]

if availability < 0.80:
    score = 0
else:
    score = (availability - 0.80) / 0.20
```

### Latency (Phase 2 - Adaptive P10-P90)
```
input:  latency in seconds
output: score ∈ [0, 1]

p10, p90 = getAdaptiveLatencyBounds()
clamped = clamp(latency, p10, p90)
score = 1 - (clamped - p10) / (p90 - p10)
```

### Sync (Phase 2 - Adaptive P10-P90)
```
input:  sync lag in seconds
output: score ∈ [0, 1]

p10, p90 = getAdaptiveSyncBounds()
clamped = clamp(syncLag, p10, p90)
score = 1 - (clamped - p10) / (p90 - p10)
```

### Stake (Square Root Scaling)
```
input:  stake, totalStake
output: score ∈ [0, 1]

ratio = stake / totalStake
score = sqrt(ratio)
```

### Composite Score
```
composite = availability * 0.3 + latency * 0.3 + sync * 0.2 + stake * 0.2
composite = max(composite, minSelectionChance)  // Floor at 0.01
composite = min(composite, 1.0)                  // Cap at 1.0
```

---

## Appendix C: Expected Distribution Calculations

### Weighted Random Selection Probability
```
P(select provider_i) = score_i / Σ(all scores)

Example with scores [0.8, 0.4, 0.2]:
  total = 0.8 + 0.4 + 0.2 = 1.4
  P(0.8) = 0.8/1.4 = 57.14%
  P(0.4) = 0.4/1.4 = 28.57%
  P(0.2) = 0.2/1.4 = 14.29%
```

### Exponential Decay
```
decayFactor = exp(-(ln(2) * timeDiff) / halfLife)

After 1 half-life:  exp(-ln(2)) = 0.5     (50% weight)
After 2 half-lives: exp(-2*ln(2)) = 0.25  (25% weight)
After 3 half-lives: exp(-3*ln(2)) = 0.125 (12.5% weight)
```
