#!/bin/bash -x

GTH_HOST=GET_GTH_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING GTH PROVIDERS :::" $GTH_HOST

# SINGLE PROXY
MOCK_PORT=2002
go run ./testutil/e2e/proxy/. $GTH_HOST  -p $MOCK_PORT -cache -id eth &

echo " ::: RUNNING GTH PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2121 http://0.0.0.0:$MOCK_PORT/eth/ GTH1 jsonrpc --from servicer1 &
lavad server 127.0.0.1 2122 http://0.0.0.0:$MOCK_PORT/eth/ GTH1 jsonrpc --from servicer2 &
lavad server 127.0.0.1 2123 http://0.0.0.0:$MOCK_PORT/eth/ GTH1 jsonrpc --from servicer3 &
lavad server 127.0.0.1 2124 http://0.0.0.0:$MOCK_PORT/eth/ GTH1 jsonrpc --from servicer4 &
lavad server 127.0.0.1 2125 http://0.0.0.0:$MOCK_PORT/eth/ GTH1 jsonrpc --from servicer5 &
lavad portal_server 127.0.0.1 3339 GTH1 jsonrpc --from user1

echo " ::: GTH PROVIDERS DONE! :::"