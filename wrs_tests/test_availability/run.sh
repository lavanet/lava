#!/bin/bash
set -euo pipefail

# Availability impact test: availability 1.0 / 0.9 / 0.8, balanced weights.
bash ../_framework/run.sh

