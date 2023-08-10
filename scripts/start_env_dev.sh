#!/bin/bash 

if [ -n "$1" ]; then
    killall lavad lava-protocol
    make install-all
    lavad start 2>&1 | grep -e "- address:" | grep -e lava_ | grep -e ERR_
else
    echo "dont use this script with vscode debugger"
    killall lavad lava-protocol
    make install-all
    ./scripts/init_chain.sh 2>&1 | grep -e "- address:" | grep -e lava_ | grep -e ERR_
fi
