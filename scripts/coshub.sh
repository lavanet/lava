#!/bin/bash -x

COSHUB_RPC=GET_COS5_VARIBLE_FROM_ENV
COSHUB_REST=GET_URL_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING COS5 PROVIDERS :::" $COS5_HOST $COS5_URL_PATH

# SINGLE PROXY
MOCK_PORT_A=2131
MOCK_PORT_B=2141
go run ./testutil/e2e/proxy/. $COSHUB_REST -p $MOCK_PORT_A -cache -id coshub_rest &
go run ./testutil/e2e/proxy/. $COSHUB_RPC -p $MOCK_PORT_B -cache -id coshub_rpc  &

echo " ::: RUNNING COS5 PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2331 http://0.0.0.0:$MOCK_PORT_A/ COS5 rest --from servicer1 &
lavad server 127.0.0.1 2332 http://0.0.0.0:$MOCK_PORT_A/ COS5 rest --from servicer2 &
lavad server 127.0.0.1 2333 http://0.0.0.0:$MOCK_PORT_A/ COS5 rest --from servicer3 &
lavad server 127.0.0.1 2341 http://0.0.0.0:$MOCK_PORT_B/ COS5 tendermintrpc --from servicer1 &
lavad server 127.0.0.1 2342 http://0.0.0.0:$MOCK_PORT_B/ COS5 tendermintrpc --from servicer2 &
lavad server 127.0.0.1 2343 http://0.0.0.0:$MOCK_PORT_B/ COS5 tendermintrpc --from servicer3 


echo " ::: COS5 PROVIDERS DONE! :::"