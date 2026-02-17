# WRS Algorithm - Enhanced Logging and Metrics

## Summary

Added comprehensive logging and metrics to the Weighted Random Selection (WRS) algorithm to enable detailed analysis of provider scores, parameter contributions, and selection probabilities.

---

## Changes Made

### 1. Enhanced Score Calculation Logging

**Location:** `protocol/provideroptimizer/weighted_selector.go` - `CalculateScore()` function

**New Log Output:** `Provider score calculation breakdown`

**Fields Logged:**
- **Provider:** `provider` - Provider address
- **Raw Values:**
  - `raw_availability` - Raw availability (0-1)
  - `raw_latency_sec` - Raw latency in seconds
  - `raw_sync_sec` - Raw sync lag in seconds
  - `raw_stake` - Raw stake amount (e.g., "1ulava")
- **Normalized Scores (0-1):**
  - `normalized_availability` - After Phase 1 rescaling
  - `normalized_latency` - After Phase 2 P10-P90 normalization
  - `normalized_sync` - After Phase 2 P10-P90 normalization
  - `normalized_stake` - After square root scaling
- **Weights:**
  - `availability_weight` - Weight for availability (default: 0.3)
  - `latency_weight` - Weight for latency (default: 0.3)
  - `sync_weight` - Weight for sync (default: 0.2)
  - `stake_weight` - Weight for stake (default: 0.2)
- **Weighted Contributions:**
  - `availability_contribution` = `normalized_availability × availability_weight`
  - `latency_contribution` = `normalized_latency × latency_weight`
  - `sync_contribution` = `normalized_sync × sync_weight`
  - `stake_contribution` = `normalized_stake × stake_weight`
- **Final Score:**
  - `composite_score` - Sum of all contributions (0-1)
  - `min_chance_adjusted` - Whether score was adjusted to minimum threshold

**Example Log:**
```
DBG Provider score calculation breakdown 
    provider=provider-2220-tendermintrpc 
    raw_availability=1.0 
    raw_latency_sec=0.052 
    raw_sync_sec=0.0 
    raw_stake=1ulava 
    normalized_availability=1.0 
    normalized_latency=0.95 
    normalized_sync=1.0 
    normalized_stake=0.333 
    availability_weight=0.3 
    latency_weight=0.3 
    sync_weight=0.2 
    stake_weight=0.2 
    availability_contribution=0.3 
    latency_contribution=0.285 
    sync_contribution=0.2 
    stake_contribution=0.0666 
    composite_score=0.8516 
    min_chance_adjusted=false
```

### 2. Enhanced Provider Selection Logging

**Location:** `protocol/provideroptimizer/weighted_selector.go` - `SelectProviderWithStats()` function

**New Log Output:** `Provider selection completed`

**Fields Logged:**
- **Selection Result:**
  - `selected_provider` - The provider that was selected
  - `selected_score` - The selected provider's composite score
  - `selected_probability_pct` - Selection probability as percentage
  - `total_score` - Sum of all provider scores
  - `random_value` - RNG value used for selection
  - `num_candidates` - Number of providers in selection pool
- **All Candidates (for each provider):**
  - `candidate_N_provider` - Provider address
  - `candidate_N_score` - Composite score
  - `candidate_N_probability_pct` - Selection probability
  - `candidate_N_availability` - Normalized availability score
  - `candidate_N_latency` - Normalized latency score
  - `candidate_N_sync` - Normalized sync score
  - `candidate_N_stake` - Normalized stake score
  - `candidate_N_composite` - Composite score (verification)

**Example Log:**
```
DBG Provider selection completed 
    selected_provider=provider-2220-tendermintrpc 
    selected_score=0.8516 
    selected_probability_pct=35.2 
    total_score=2.419 
    random_value=0.753 
    num_candidates=3 
    candidate_1_provider=provider-2220-tendermintrpc 
    candidate_1_score=0.8516 
    candidate_1_probability_pct=35.2 
    candidate_1_availability=1.0 
    candidate_1_latency=0.95 
    candidate_1_sync=1.0 
    candidate_1_stake=0.333 
    candidate_1_composite=0.8516 
    candidate_2_provider=provider-2221-tendermintrpc 
    candidate_2_score=0.8516 
    candidate_2_probability_pct=35.2 
    candidate_2_availability=1.0 
    candidate_2_latency=0.93 
    candidate_2_sync=1.0 
    candidate_2_stake=0.333 
    candidate_2_composite=0.8516 
    candidate_3_provider=provider-2222-tendermintrpc 
    candidate_3_score=0.7158 
    candidate_3_probability_pct=29.6 
    candidate_3_availability=1.0 
    candidate_3_latency=0.92 
    candidate_3_sync=0.75 
    candidate_3_stake=0.333 
    candidate_3_composite=0.7158
```

### 3. Enhanced Metrics Endpoint

**Location:** `protocol/metrics/consumer_optimizer_qos_client.go`

**Clarified Existing Fields:**

The metrics endpoint already had most fields, but they were poorly documented:

**Legacy Fields (Raw EWMA values - NOT normalized for WRS):**
```json
{
  "sync_score": 0.0,           // Raw sync lag in seconds from EWMA (lower is better)
  "availability_score": 1.0,   // Raw availability from EWMA (0-1, higher is better)
  "latency_score": 0.052,      // Raw latency in seconds from EWMA (lower is better)
  "generic_score": 0.8516      // Old composite score (deprecated)
}
```

**WRS Normalized Scores (Already existed - used for selection):**
```json
{
  "selection_availability": 1.0,   // Normalized after Phase 1 rescaling
  "selection_latency": 0.95,       // Normalized after Phase 2 P10-P90
  "selection_sync": 1.0,           // Normalized after Phase 2 P10-P90
  "selection_stake": 0.333,        // Normalized after square root scaling
  "selection_composite": 0.8516    // Final composite score
}
```

**NEW - Weighted Contributions:**
```json
{
  "availability_contribution": 0.3,     // selection_availability × 0.3
  "latency_contribution": 0.285,        // selection_latency × 0.3
  "sync_contribution": 0.2,             // selection_sync × 0.2
  "stake_contribution": 0.0666          // selection_stake × 0.2
}
```

**Complete Metrics JSON Example:**
```json
{
  "provider": "provider-2220-tendermintrpc",
  "chain_id": "LAV1",
  
  // Legacy fields (raw EWMA values - NOT normalized)
  "sync_score": 0.0,               // Raw sync lag in seconds
  "availability_score": 1.0,       // Raw availability (0-1)
  "latency_score": 0.052,          // Raw latency in seconds
  "generic_score": 0.8516,         // Old composite (deprecated)
  
  // WRS normalized scores (already existed - used for selection)
  "selection_availability": 1.0,   // Normalized after Phase 1 rescaling
  "selection_latency": 0.95,       // Normalized after Phase 2 P10-P90
  "selection_sync": 1.0,           // Normalized after Phase 2 P10-P90
  "selection_stake": 0.333,        // Normalized after square root scaling
  "selection_composite": 0.8516,   // Final composite score
  
  // NEW - Weighted contributions (what we added)
  "availability_contribution": 0.3,    // selection_availability × 0.3
  "latency_contribution": 0.285,       // selection_latency × 0.3
  "sync_contribution": 0.2,            // selection_sync × 0.2
  "stake_contribution": 0.0666,        // selection_stake × 0.2
  
  // Selection tracking (already existed)
  "selection_count": 210,
  "selection_rate": 0.35,
  "selection_qos_score": 0.8516,
  "selection_rng_value": 0.753
}
```

---

## Usage

### 1. Analyzing Logs

**Extract score breakdowns:**
```bash
grep "Provider score calculation breakdown" CONSUMER.log | grep provider-2220
```

**Extract selection events:**
```bash
grep "Provider selection completed" CONSUMER.log
```

**Verify score calculations:**
```bash
# Check that composite_score = sum of contributions
grep "Provider score calculation breakdown" CONSUMER.log | \
  awk '{for(i=1;i<=NF;i++) if($i~/availability_contribution=|latency_contribution=|sync_contribution=|stake_contribution=|composite_score=/) print $i}'
```

### 2. Analyzing Metrics Endpoint

**Query metrics:**
```bash
curl http://localhost:7779/provider_optimizer_metrics
```

**Parse with jq:**
```bash
curl -s http://localhost:7779/provider_optimizer_metrics | jq '.[] | {
  provider,
  
  # Legacy raw EWMA values
  raw_availability: .availability_score,
  raw_latency_sec: .latency_score,
  raw_sync_sec: .sync_score,
  
  # WRS normalized scores
  normalized_availability: .selection_availability,
  normalized_latency: .selection_latency,
  normalized_sync: .selection_sync,
  normalized_stake: .selection_stake,
  composite: .selection_composite,
  
  # Weighted contributions (NEW)
  contrib_avail: .availability_contribution,
  contrib_latency: .latency_contribution,
  contrib_sync: .sync_contribution,
  contrib_stake: .stake_contribution,
  
  selection_rate
}'
```

### 3. Python Analysis

**Example script to verify calculations:**
```python
import json
import requests

# Fetch metrics
response = requests.get('http://localhost:7779/provider_optimizer_metrics')
providers = response.json()

for provider in providers:
    # Extract values
    avail_contrib = provider['availability_contribution']
    latency_contrib = provider['latency_contribution']
    sync_contrib = provider['sync_contribution']
    stake_contrib = provider['stake_contribution']
    composite = provider['selection_composite']
    
    # Verify calculation
    calculated_composite = avail_contrib + latency_contrib + sync_contrib + stake_contrib
    diff = abs(composite - calculated_composite)
    
    print(f"Provider: {provider['provider']}")
    print(f"  Composite (reported): {composite:.4f}")
    print(f"  Composite (calculated): {calculated_composite:.4f}")
    print(f"  Difference: {diff:.6f}")
    print(f"  Breakdown: {avail_contrib:.4f} + {latency_contrib:.4f} + {sync_contrib:.4f} + {stake_contrib:.4f}")
    print()
```

---

## Test Verification

### Formula Verification

For each provider, verify:

**1. Normalized Scores:**
- Availability: `(raw_availability - 0.80) / (1.0 - 0.80)` (Phase 1)
- Latency: `1 - (clamp(raw_latency, P10, P90) - P10) / (P90 - P10)` (Phase 2)
- Sync: `1 - (clamp(raw_sync, P10, P90) - P10) / (P90 - P10)` (Phase 2)
- Stake: `sqrt(provider_stake / total_stake)` (Square root scaling)

**2. Contributions:**
- `availability_contribution = normalized_availability × 0.3`
- `latency_contribution = normalized_latency × 0.3`
- `sync_contribution = normalized_sync × 0.2`
- `stake_contribution = normalized_stake × 0.2`

**3. Composite Score:**
- `composite_score = availability_contribution + latency_contribution + sync_contribution + stake_contribution`

**4. Selection Probability:**
- `probability = composite_score / sum_of_all_composite_scores`

### Distribution Verification

After running a test with N requests:

**1. Count selections from logs:**
```bash
grep "Provider selection completed" CONSUMER.log | \
  grep -o 'selected_provider=[^ ]*' | \
  sort | uniq -c
```

**2. Compare with expected probabilities from metrics:**
```bash
curl -s http://localhost:7779/provider_optimizer_metrics | \
  jq '.[] | {provider, selection_rate, selection_count}'
```

**3. Verify distribution matches probabilities** (within statistical tolerance)

---

## Benefits

### For Testing
1. **Verify each normalization step** - See raw → normalized transformation
2. **Verify weighting** - See contribution of each parameter
3. **Verify composite calculation** - Sum of contributions
4. **Verify selection probabilities** - Score / total
5. **Track actual selections** - Compare expected vs actual

### For Debugging
1. **Identify miscalculations** - Compare logs with expected formulas
2. **Spot outliers** - Unusual raw values or normalized scores
3. **Validate Phase 2** - Check P10/P90 bounds are being used
4. **Trace selection logic** - See RNG value and cumulative probabilities

### For Monitoring
1. **Real-time score tracking** - Via metrics endpoint
2. **Distribution analysis** - Selection rates over time
3. **Parameter impact** - Which parameters dominate selection
4. **Network health** - Aggregate QoS metrics

---

## Example Analysis Workflow

### 1. Run Test
```bash
./scripts/pre_setups/run_test3_sync_impact.sh
```

### 2. Analyze Score Calculations
```bash
# Extract all score breakdowns
grep "Provider score calculation breakdown" testutil/debugging/logs/CONSUMER.log > scores.log

# Check provider 2220
grep "provider-2220" scores.log | tail -5
```

### 3. Analyze Selection Events
```bash
# Extract all selections
grep "Provider selection completed" testutil/debugging/logs/CONSUMER.log > selections.log

# Count selections per provider
grep -o 'selected_provider=[^ ]*' selections.log | sort | uniq -c
```

### 4. Verify with Metrics
```bash
# Query current state
curl -s http://localhost:7779/provider_optimizer_metrics | jq '.'

# Extract selection rates
curl -s http://localhost:7779/provider_optimizer_metrics | \
  jq '.[] | {provider, composite: .selection_composite, rate: .selection_rate}'
```

### 5. Calculate Expected vs Actual
```python
import json
import requests

# Get metrics
metrics = requests.get('http://localhost:7779/provider_optimizer_metrics').json()

# Calculate expected probabilities
total_score = sum(p['selection_composite'] for p in metrics)
for p in metrics:
    expected_prob = p['selection_composite'] / total_score
    actual_rate = p['selection_rate']
    print(f"{p['provider']}: Expected={expected_prob:.1%}, Actual={actual_rate:.1%}")
```

---

## Files Modified

1. **protocol/provideroptimizer/weighted_selector.go**
   - Added comprehensive logging to `CalculateScore()`
   - Enhanced selection logging in `SelectProviderWithStats()`
   - Updated `CalculateProviderScores()` to populate new metrics fields

2. **protocol/metrics/consumer_optimizer_qos_client.go**
   - Added `RawAvailability`, `RawLatencySeconds`, `RawSyncSeconds`, `RawStake` fields
   - Added `AvailabilityContribution`, `LatencyContribution`, `SyncContribution`, `StakeContribution` fields
   - Updated `appendOptimizerQoSReport()` to include new fields

---

## Backward Compatibility

All changes are **backward compatible**:
- Existing log parsers will continue to work
- New log fields are additions, not replacements
- Existing metrics fields are preserved
- New metrics fields are additions
- JSON parsers that ignore unknown fields will work unchanged

---

## Next Steps

1. **Test the enhanced logging** - Run Test 3 and verify logs contain all fields
2. **Update Python analysis scripts** - Use new metrics fields for more accurate analysis
3. **Create dashboard** - Visualize scores, contributions, and selection rates
4. **Document test results** - Update test reports with enhanced data
