#!/bin/bash 

if [ -n "$1" ]; then
    killall lavad lavap
    make install-all
    lavad start 
else
    echo "dont use this script with vscode debugger"
    killall lavad lavap
    make install-all
    ./scripts/init_chain.sh 
fi
