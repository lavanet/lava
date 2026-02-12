#!/bin/bash
set -euo pipefail

# Latency impact test: delays 10ms / 50ms / 100ms, balanced weights.
bash ../_framework/run.sh

