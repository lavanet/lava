#!/bin/bash -x

ETH_HOST=GET_ETH_VARIBLE_FROM_ENV
ETH_URL_PATH=GET_URL_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo ""
echo " ::: STARTING ETH PROVIDERS :::" $ETH_RPC_HTTP $URL_PATH

# SINGLE PROXY
MOCK_PORT=2001
go run ./testutil/e2e/proxy/. $ETH_HOST  -p $MOCK_PORT -cache -id eth &

# Multi Port Proxy
# sh ./scripts/mock_proxy_eth.sh &
# sleep 2

# echo " ::: DONE ETHMOCK PROXY ::: "

echo " ::: RUNNING ETH PROVIDERS :::"
# SINGLE MOCK PROXY
lavad server 127.0.0.1 2221 http://0.0.0.0:$MOCK_PORT/$ETH_URL_PATH ETH1 jsonrpc --from servicer1 &
lavad server 127.0.0.1 2222 http://0.0.0.0:$MOCK_PORT/$ETH_URL_PATH ETH1 jsonrpc --from servicer2 &
lavad server 127.0.0.1 2223 http://0.0.0.0:$MOCK_PORT/$ETH_URL_PATH ETH1 jsonrpc --from servicer3 &
lavad server 127.0.0.1 2224 http://0.0.0.0:$MOCK_PORT/$ETH_URL_PATH ETH1 jsonrpc --from servicer4 &
lavad server 127.0.0.1 2225 http://0.0.0.0:$MOCK_PORT/$ETH_URL_PATH ETH1 jsonrpc --from servicer5 

# Multi Port Proxy
# lavad server 127.0.0.1 2221 http://0.0.0.0:2001/$ETH_URL_PATH ETH1 jsonrpc --from servicer1 &
# lavad server 127.0.0.1 2222 http://0.0.0.0:2002/$ETH_URL_PATH ETH1 jsonrpc --from servicer2 &
# lavad server 127.0.0.1 2223 http://0.0.0.0:2003/$ETH_URL_PATH ETH1 jsonrpc --from servicer3 &
# lavad server 127.0.0.1 2224 http://0.0.0.0:2004/$ETH_URL_PATH ETH1 jsonrpc --from servicer4 &
# lavad server 127.0.0.1 2225 http://0.0.0.0:2005/$ETH_URL_PATH ETH1 jsonrpc --from servicer5 

# NO MOCK PROXY
# lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 &
# lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2 &
# lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3 &
# lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4 &
# lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5 

echo " ::: ETH PROVIDERS DONE! :::"