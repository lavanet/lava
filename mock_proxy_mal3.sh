echo ""
echo " ::: STARTING PROXY SERVERS :::"
killall proxy
go run ./testutil/e2e/proxy/. 2001 mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2002 mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2003 mainnet.infura.io 1 &
go run ./testutil/e2e/proxy/. 2004 mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2005 mainnet.infura.io 

echo " ::: DONE MOCK PROXY ::: "