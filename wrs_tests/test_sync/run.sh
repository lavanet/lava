#!/bin/bash
set -euo pipefail

# Sync impact test: gap_blocks 0/1/2, balanced weights (set by defaults in framework unless overridden).
bash ../_framework/run.sh

