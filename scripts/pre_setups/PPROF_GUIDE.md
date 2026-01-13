# Pprof Memory Profiling Guide

This guide explains how to use pprof to monitor memory usage of the Lava providers.

## Available Pprof Endpoints

After running `init_lava_static_and_backup_provider_eth.sh`, the following pprof endpoints are available:

- **Provider 1**: http://127.0.0.1:6060/debug/pprof/
- **Provider 2**: http://127.0.0.1:6061/debug/pprof/
- **Provider 3**: http://127.0.0.1:6062/debug/pprof/
- **SmartRouter**: http://127.0.0.1:6063/debug/pprof/

## Common Pprof Commands

### 1. View Available Profiles (in Browser)

Open any of the pprof URLs in your browser to see available profiles:
```bash
open http://127.0.0.1:6060/debug/pprof/
```

### 2. Capture Heap Memory Profile

Capture a snapshot of the current memory allocation:
```bash
# Provider 1
curl http://127.0.0.1:6060/debug/pprof/heap > provider1_heap.prof

# Provider 2
curl http://127.0.0.1:6061/debug/pprof/heap > provider2_heap.prof

# Provider 3
curl http://127.0.0.1:6062/debug/pprof/heap > provider3_heap.prof
```

### 3. Analyze Heap Profile Interactively

View the profile in your browser with an interactive graph:
```bash
go tool pprof -http=:8080 provider1_heap.prof
```

This will open a browser window at http://localhost:8080 with various views:
- **Top** - Top memory consumers
- **Graph** - Visual graph of memory allocation
- **Flame Graph** - Hierarchical view of allocations
- **Peek** - Source code view
- **Source** - Annotated source code

### 4. Compare Two Heap Profiles (Memory Leak Detection)

Capture two profiles at different times and compare them:
```bash
# Capture baseline
curl http://127.0.0.1:6060/debug/pprof/heap > baseline.prof

# Wait some time (e.g., run some load tests)
sleep 300

# Capture second snapshot
curl http://127.0.0.1:6060/debug/pprof/heap > after_load.prof

# Compare (shows what memory increased)
go tool pprof -http=:8080 -base baseline.prof after_load.prof
```

### 5. Capture Goroutine Profile

See all running goroutines (useful for goroutine leaks):
```bash
curl http://127.0.0.1:6060/debug/pprof/goroutine > provider1_goroutine.prof
go tool pprof -http=:8080 provider1_goroutine.prof
```

### 6. View Live Memory Allocation

Get a text summary of top memory allocators:
```bash
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
# In the pprof prompt, type:
# > top10
# > list <function_name>
```

### 7. Monitor Memory Over Time

Use a script to periodically capture profiles:
```bash
#!/bin/bash
mkdir -p memory_profiles
for i in {1..60}; do
  timestamp=$(date +%Y%m%d_%H%M%S)
  curl http://127.0.0.1:6060/debug/pprof/heap > memory_profiles/provider1_${timestamp}.prof
  echo "Captured profile $i at $timestamp"
  sleep 60  # Capture every minute
done
```

### 8. Get Allocation Statistics

View detailed allocation statistics:
```bash
curl http://127.0.0.1:6060/debug/pprof/allocs > provider1_allocs.prof
go tool pprof -http=:8080 provider1_allocs.prof
```

## Understanding the Output

### Key Metrics

- **inuse_space**: Amount of memory currently in use
- **inuse_objects**: Number of objects currently in use
- **alloc_space**: Total amount of memory allocated (including freed)
- **alloc_objects**: Total number of objects allocated (including freed)

### Command Line Options

```bash
# Focus on current memory usage (inuse)
go tool pprof -http=:8080 -sample_index=inuse_space provider1_heap.prof

# Focus on total allocations (useful for finding allocation hotspots)
go tool pprof -http=:8080 -sample_index=alloc_space provider1_heap.prof
```

## Quick Memory Leak Detection Workflow

1. **Establish baseline** before load:
   ```bash
   curl http://127.0.0.1:6060/debug/pprof/heap > baseline.prof
   ```

2. **Run your workload** for a period of time

3. **Capture profile after load**:
   ```bash
   curl http://127.0.0.1:6060/debug/pprof/heap > after.prof
   ```

4. **Compare profiles**:
   ```bash
   go tool pprof -http=:8080 -base baseline.prof after.prof
   ```

5. **Look for functions with large positive deltas** - these are likely memory leaks

## All Available Endpoints

For each provider, these endpoints are available:

- `/debug/pprof/` - Index page
- `/debug/pprof/heap` - Heap memory profile
- `/debug/pprof/goroutine` - Goroutine profile
- `/debug/pprof/threadcreate` - Thread creation profile
- `/debug/pprof/block` - Block profile
- `/debug/pprof/mutex` - Mutex contention profile
- `/debug/pprof/allocs` - All memory allocations profile
- `/debug/pprof/profile?seconds=30` - CPU profile (30 seconds)

## Tips

1. **Heap profiles show sampled data** - not every allocation is recorded, but the sampling is representative
2. **Use `-base` flag** to find memory leaks by comparing two profiles
3. **Check both `inuse_space` and `alloc_space`** - the former shows current usage, the latter shows total allocations
4. **Goroutine profiles** can reveal goroutine leaks which often accompany memory leaks
5. **Run profiles multiple times** to ensure results are consistent

## Example: Full Investigation

```bash
# 1. Capture baseline
curl http://127.0.0.1:6060/debug/pprof/heap > baseline.prof
curl http://127.0.0.1:6060/debug/pprof/goroutine > baseline_goroutine.prof

# 2. Run load for 10 minutes
# (your load testing here)

# 3. Capture after load
curl http://127.0.0.1:6060/debug/pprof/heap > after.prof
curl http://127.0.0.1:6060/debug/pprof/goroutine > after_goroutine.prof

# 4. Compare memory
go tool pprof -http=:8081 -base baseline.prof after.prof

# 5. Compare goroutines
go tool pprof -http=:8082 -base baseline_goroutine.prof after_goroutine.prof
```
















