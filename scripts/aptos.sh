#!/bin/bash -x

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
APTOS_REST=GET_APTOS_VARIABLE_FROM_ENV
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING FTM PROVIDERS :::" $APTOS_REST 

# SINGLE PROXY
MOCK_PORT=1993
go run ./testutil/e2e/proxy/. $APTOS_REST  -p $MOCK_PORT -cache -id aptos &

echo " ::: RUNNING APTOS PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2281 http://0.0.0.0:$MOCK_PORT/aptos/http APT1 rest --from servicer1 &
lavad server 127.0.0.1 2282 http://0.0.0.0:$MOCK_PORT/aptos/http APT1 rest --from servicer2 &
lavad server 127.0.0.1 2283 http://0.0.0.0:$MOCK_PORT/aptos/http APT1 rest --from servicer3 &
lavad portal_server 127.0.0.1 3336 APT1 rest --from user4

echo " ::: APTOS PROVIDERS DONE! :::"
