#!/bin/bash -x

OSMO_HOST=GET_OSMO_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh 

echo ""
echo " ::: STARTING OSMOSIS PROVIDERS :::" $OSMO_HOST

    # SINGLE PROXY
MOCK_PORT_A=2031
MOCK_PORT_B=2041
go run ./testutil/e2e/proxy/. $OSMO_HOST -p $MOCK_PORT_A -cache -id osmosis_rest  &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p $MOCK_PORT_B -cache -id osmosis_rpc  &

# Multi Port Proxy
# sh ./mock_proxy_osmosis.sh &


echo " ::: RUNNING OSMOSIS PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2231 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer1 &
lavad server 127.0.0.1 2232 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer2 &
lavad server 127.0.0.1 2233 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer3 &
lavad server 127.0.0.1 2241 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer1 &
lavad server 127.0.0.1 2242 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer2 &
lavad server 127.0.0.1 2243 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer3 

# Multi Port Proxy
# lavad server 127.0.0.1 2231 http://0.0.0.0:2031/rest/ COS3 rest --from servicer1 &
# lavad server 127.0.0.1 2232 http://0.0.0.0:2032/rest/ COS3 rest --from servicer2 &
# lavad server 127.0.0.1 2233 http://0.0.0.0:2033/rest/ COS3 rest --from servicer3 &
# lavad server 127.0.0.1 2241 http://0.0.0.0:2041/rpc/ COS3 tendermintrpc --from servicer1 &
# lavad server 127.0.0.1 2242 http://0.0.0.0:2042/rpc/ COS3 tendermintrpc --from servicer2 &
# lavad server 127.0.0.1 2243 http://0.0.0.0:2043/rpc/ COS3 tendermintrpc --from servicer3 

# NO MOCK PROXY
# lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest --from servicer1 &
# lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest --from servicer2 &
# lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest --from servicer3 &
# lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc --from servicer1 &
# lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc --from servicer2 &
# lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc --from servicer3 

echo " ::: OSMOSIS PROVIDERS DONE! :::"