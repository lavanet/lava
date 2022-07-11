#!/bin/bash
__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. ${__dir}/scripts/variables.sh

echo ""
echo " ::: STARTING PROXY SERVERS :::"
killall proxy
go run ./testutil/e2e/proxy/. 2031 $OSMO_HOST &
go run ./testutil/e2e/proxy/. 2032 $OSMO_HOST &
go run ./testutil/e2e/proxy/. 2033 $OSMO_HOST 1 &
go run ./testutil/e2e/proxy/. 2041 $OSMO_HOST &
go run ./testutil/e2e/proxy/. 2042 $OSMO_HOST &
go run ./testutil/e2e/proxy/. 2043 $OSMO_HOST 1

echo " ::: DONE MOCK PROXY ::: "


# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
# $OSMO_HOST
