echo ""
echo " ::: STARTING PROXY SERVERS :::"
# killall proxy
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p 2001 -id eth &
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p 2002 -id eth &
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p 2003 -id eth &
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p 2004 -id eth &
go run ./testutil/e2e/proxy/. https://mainnet.infura.io -p 2005 -id eth 

echo " ::: DONE MOCK PROXY ::: "