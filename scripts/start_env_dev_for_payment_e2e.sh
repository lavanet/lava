#!/bin/bash 

if [ -n "$1" ]; then
    killall lavad lavap
    make install-all LAVA_BUILD_OPTIONS="debug_payment_e2e"
    lavad start 
else
    echo "dont use this script with vscode debugger"
    killall lavad lavap
    make install-all LAVA_BUILD_OPTIONS="debug_payment_e2e"
    ./scripts/init_chain.sh debug
fi
