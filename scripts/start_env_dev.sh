#!/bin/bash 
echo "dont use this script with vscode debugger"
killall lava-protocol
make build-protocol
echo "deleting transcation database (rm -rf .storage)" # we cant keep the database when creating new addresses with ignite
rm -rf .storage
rm -rf build
ignite chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !