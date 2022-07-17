#!/bin/bash -x

OSMO_HOST=GET_OSMO_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh 

echo ""
echo " ::: STARTING PROXY SERVERS :::"
# killall proxy
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2031 -cache -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2032 -cache -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2033 -cache -id osmosis_rest &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2041 -cache -id osmosis_rpc &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2042 -cache -id osmosis_rpc &
go run ./testutil/e2e/proxy/. $OSMO_HOST -p 2043 -cache -id osmosis_rpc 

echo " ::: DONE MOCK PROXY ::: "


# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
