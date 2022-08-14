#!/bin/bash -x

FTM_HOST=GET_FTM_VARIBLE_FROM_ENV
FTM_URL_PATH=GET_URL_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING FTM PROVIDERS :::" $FTM_HOST $FTM_URL_PATH

# SINGLE PROXY
MOCK_PORT=2001
go run ./testutil/e2e/proxy/. $FTM_HOST  -p $MOCK_PORT -cache -id ftm &

echo " ::: RUNNING FTM PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2251 http://0.0.0.0:$MOCK_PORT/$FTM_URL_PATH FTM250 jsonrpc --from servicer1 &
lavad server 127.0.0.1 2252 http://0.0.0.0:$MOCK_PORT/$FTM_URL_PATH FTM250 jsonrpc --from servicer2 &
lavad server 127.0.0.1 2253 http://0.0.0.0:$MOCK_PORT/$FTM_URL_PATH FTM250 jsonrpc --from servicer3 &
lavad server 127.0.0.1 2254 http://0.0.0.0:$MOCK_PORT/$FTM_URL_PATH FTM250 jsonrpc --from servicer4 &
lavad server 127.0.0.1 2255 http://0.0.0.0:$MOCK_PORT/$FTM_URL_PATH FTM250 jsonrpc --from servicer5 

echo " ::: FTM PROVIDERS DONE! :::"