#!/bin/bash -x

OSMO_HOST=GET_OSMO_VARIBLE_FROM_ENV
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh 

echo ""
echo " ::: STARTING PROXY SERVERS :::"
# killall proxy
go run ./testutil/e2e/proxy/. $ETH_HOST -p 2001 -cache -id eth &
go run ./testutil/e2e/proxy/. $ETH_HOST -p 2002 -cache -id eth &
go run ./testutil/e2e/proxy/. $ETH_HOST -p 2003 -cache -id eth &
go run ./testutil/e2e/proxy/. $ETH_HOST -p 2004 -cache -id eth &
go run ./testutil/e2e/proxy/. $ETH_HOST -p 2005 -cache -id eth 

echo " ::: DONE MOCK PROXY ::: "