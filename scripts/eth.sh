#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/variables.sh

echo ""
echo " ::: STARTING ETH PROVIDERS :::"

# SINGLE PROXY
MOCK_PORT=2001
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p $MOCK_PORT -cache -id eth &

# Multi Port Proxy
# sh ./mock_proxy_eth.sh &

echo " ::: DONE ETHMOCK PROXY ::: "

echo " ::: RUNNING ETH PROVIDERS :::"
# SINGLE MOCK PROXY
go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 http://0.0.0.0:$MOCK_PORT/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer1 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 http://0.0.0.0:$MOCK_PORT/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer2 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 http://0.0.0.0:$MOCK_PORT/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer3 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 http://0.0.0.0:$MOCK_PORT/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer4 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 http://0.0.0.0:$MOCK_PORT/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer5 

# Multi Port Proxy
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 http://0.0.0.0:2001/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 http://0.0.0.0:2002/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 http://0.0.0.0:2003/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer3 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 http://0.0.0.0:2004/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer4 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 http://0.0.0.0:2005/v3/3755a1321ab24f938589412403c46455 ETH1 jsonrpc --from servicer5 

# NO MOCK PROXY
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5 

echo " ::: ETH PROVIDERS DONE! :::"