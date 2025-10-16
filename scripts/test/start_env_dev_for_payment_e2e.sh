#!/bin/bash

killall lavad lavap
make install-all LAVA_BUILD_OPTIONS="debug_payment_e2e"

if [ -n "$1" ]; then
    # With argument: start lavad directly (accounts should already exist)
    lavad start
else
    # Without argument: initialize chain first, then start lavad
    echo "Initializing chain for payment E2E test"
    ./scripts/init_chain.sh debug
    lavad start
fi
