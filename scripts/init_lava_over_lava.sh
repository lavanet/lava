#!/bin/bash

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
LOGS_DIR=${__dir}/../testutil/debugging/logs

PROVIDER4_LISTENER="127.0.0.1:2224"
CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"
GASPRICE="0.000000001ulava"

lavad tx pairing bulk-stake-provider LAV1 $PROVIDERSTAKE "$PROVIDER4_LISTENER,1" 1 -y --from servicer4 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lava-protocol rpcprovider \
$PROVIDER4_LISTENER LAV1 rest http://127.0.0.1:3360/1 \
$PROVIDER4_LISTENER LAV1 tendermintrpc http://127.0.0.1:3361/1 \
$PROVIDER4_LISTENER LAV1 grpc 127.0.0.1:3362 \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer4 2>&1 | tee $LOGS_DIR/PROVIDER4.log

