echo ""
echo " ::: STARTING PROXY SERVERS :::"
# killall proxy
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2031 -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2032 -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2033 -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2041 -id osmosis_rpc &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2042 -id osmosis_rpc &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2043 -id osmosis_rpc 

echo " ::: DONE MOCK PROXY ::: "


# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
