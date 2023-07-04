#!/bin/bash 
killall lava-protocol
make build-protocol
rm -rf build
ignite chain serve -v -r 2>&1 | grep -e lava_ -e ERR_ -e STARPORT] -e !