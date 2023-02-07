#!/bin/bash -x

JUNO_RPC=GET_JUN1_VARIBLE_FROM_ENV
JUNO_REST=GET_JUN1_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING JUN1 PROVIDERS :::" $JUN1_HOST $JUN1_URL_PATH

# SINGLE PROXY
MOCK_PORT_A=2431
MOCK_PORT_B=2441
go run ./testutil/e2e/proxy/. $JUNO_REST -p $MOCK_PORT_A -cache -id juno_rest &
go run ./testutil/e2e/proxy/. $JUNO_RPC -p $MOCK_PORT_B -cache -id juno_rpc  &

echo " ::: RUNNING JUN1 PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2371 http://0.0.0.0:$MOCK_PORT_A/ JUN1 rest --from servicer1 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2372 http://0.0.0.0:$MOCK_PORT_A/ JUN1 rest --from servicer2 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2373 http://0.0.0.0:$MOCK_PORT_A/ JUN1 rest --from servicer3 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2361 http://0.0.0.0:$MOCK_PORT_B/ JUN1 tendermintrpc --from servicer1 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2362 http://0.0.0.0:$MOCK_PORT_B/ JUN1 tendermintrpc --from servicer2 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2363 http://0.0.0.0:$MOCK_PORT_B/ JUN1 tendermintrpc --from servicer3 --geolocation 1 --log_level debug 


echo " ::: JUN1 PROVIDERS DONE! :::"
