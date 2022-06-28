echo ""
echo " ::: STARTING OSMOSIS PROVIDERS :::"

# SINGLE PROXY
MOCK_PORT_A=2031
MOCK_PORT_B=2041
go run ./testutil/e2e/proxy/. DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235 -p $MOCK_PORT_A -cache -id osmosis_rest  &
go run ./testutil/e2e/proxy/. DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235 -p $MOCK_PORT_B -cache -id osmosis_rpc  &

# Multi Port Proxy
# sh ./mock_proxy_osmosis.sh &


echo " ::: RUNNING OSMOSIS PROVIDERS :::"
# SINGLE MOCK PROXY
go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer1 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer2 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://0.0.0.0:$MOCK_PORT_A/rest/ COS3 rest --from servicer3 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer1 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer2 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://0.0.0.0:$MOCK_PORT_B/rpc/ COS3 tendermintrpc --from servicer3 

# Multi Port Proxy
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://0.0.0.0:2031/rest/ COS3 rest --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://0.0.0.0:2032/rest/ COS3 rest --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://0.0.0.0:2033/rest/ COS3 rest --from servicer3 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://0.0.0.0:2041/rpc/ COS3 tendermintrpc --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://0.0.0.0:2042/rpc/ COS3 tendermintrpc --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://0.0.0.0:2043/rpc/ COS3 tendermintrpc --from servicer3 

# NO MOCK PROXY
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer3 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer3 

echo " ::: OSMOSIS PROVIDERS DONE! :::"