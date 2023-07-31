#!/bin/sh
# vim:sw=4:ts=4:et

set -e

info() {
    echo "INF: $@"
}

debug() {
    echo "DBG: $@"
}

error() {
    echo "ERR: $@"
    exit 1
}

check_env_vars() {
    # no need to check LAVA_USER and LAVA_ADDRESS: see check_lava_addr()

    env_vars="LAVA_KEYRING \
        LAVA_STAKE_AMOUNT \
        LAVA_GEOLOCATION \
        LAVA_RPC_NODE \
        LAVA_CHAIN_ID \
        LAVA_GAS_MODE \
        LAVA_GAS_ADJUST \
        LAVA_GAS_PRICE \
        LAVA_LISTEN_IP \
        LAVA_PORTAL_PORT \
        LAVA_RELAY_ENDPOINT \
        LAVA_RELAY_NODE_URL \
        LAVA_RELAY_CHAIN_ID \
        LAVA_RELAY_IFACE \
        LAVA_LOG_LEVEL \
    "

    for ev in $env_vars; do
        eval "v=\$$ev"
        test -z "${v}" && errmsg="${errmsg}     ${ev}\n"
    done

    if [ -n "$errmsg" ]; then
        error "some env variables not defined:\n${errmsg%%\\n}"
    fi
}

check_lava_addr() {
    lavad keys list --keyring-backend "${LAVA_KEYRING}"

    if [ -z "${LAVA_ADDRESS}" ]; then
        LAVA_ADDRESS=$(lavad keys show "${LAVA_USER}" --keyring-backend "${LAVA_KEYRING}" | \
            grep address | awk '{print $2}')
  
        if [ -z "${LAVA_ADDRESS}" ]; then
            error "unable to fetch the user's lava address"
        fi
    else
        lavad keys list --keyring-backend "${LAVA_KEYRING}" | \
            grep -q ${LAVA_ADDRESS} || \
	    error "unable to find the requested lava address"
    fi
}

provider_staked_amount() {
    lavad query pairing providers \
        "${LAVA_RELAY_CHAIN_ID}" \
        --node "${LAVA_RPC_NODE}" \
        --chain-id "${LAVA_CHAIN_ID}" \
        | sed -n '/Staked Providers:/{n;p}' \
        | grep "${LAVA_ADDRESS}" \
        | sed 's/^.*{\([0-9]*ulava\).*$/\1/' 
}

stake_provider() {
    info "staking provider - this may take a while"
    lavad tx pairing stake-provider -y \
        "${LAVA_RELAY_CHAIN_ID}" \
        "${LAVA_STAKE_AMOUNT}" \
        "${LAVA_RELAY_ENDPOINT}" \
        "${LAVA_GEOLOCATION}" \
        --from "${LAVA_ADDRESS}" \
        --provider-moniker "dummyMoniker" \
        --node "${LAVA_RPC_NODE}" \
        --chain-id "${LAVA_CHAIN_ID}" \
        --keyring-backend "${LAVA_KEYRING}" \
        --gas-adjustment "${LAVA_GAS_ADJUST}" \
        --gas-prices "${LAVA_GAS_PRICE}" \
        --gas "${LAVA_GAS_MODE}" \
        --log_level "${LAVA_LOG_LEVEL}" ||
            error "unable to stake provider"
}

# check sanity of env vars
check_env_vars

# check (and maybe get) lava address
check_lava_addr

# check that provider is staked with right amount
stake_amount=$(provider_staked_amount)

if [ -z $stake_amount ]; then
    info "provider not staked: staking provider"
    stake_provider
elif [ $stake_amount -lt ${LAVA_STAKE_AMOUNT} ]; then
    info "provider staked amount to small: increasing amount"
    stake_provider
fi

debug "starting provider server"

exec lavad portal_server \
    "${LAVA_LISTEN_IP}" \
    "${LAVA_PORTAL_PORT}" \
    "${LAVA_RELAY_NODE_URL}" \
    "${LAVA_RELAY_CHAIN_ID}" \
    "${LAVA_RELAY_IFACE}" \
    --from "${LAVA_ADDRESS}" \
    --node "${LAVA_RPC_NODE}" \
    --chain-id "${LAVA_CHAIN_ID}" \
    --geolocation "${LAVA_GEOLOCATION}" \
    --log_level "${LAVA_LOG_LEVEL}" ||
        error "unable to start provider"

