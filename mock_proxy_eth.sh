echo ""
echo " ::: STARTING PROXY SERVERS :::"
killall proxy
go run ./testutil/e2e/proxy/. 2001 https://mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2002 https://mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2003 https://mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2004 https://mainnet.infura.io &
go run ./testutil/e2e/proxy/. 2005 https://mainnet.infura.io 

echo " ::: DONE MOCK PROXY ::: "