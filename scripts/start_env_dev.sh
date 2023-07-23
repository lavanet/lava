#!/bin/bash 

if [ -n "$1" ]; then # some systems dont support the log parsing.
    killall lava-protocol
    make build-protocol
    ignite chain build; lavad start
else
    echo "dont use this script with vscode debugger"
    killall lava-protocol
    make build-protocol
    rm -rf build
    ignite chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !  
fi
