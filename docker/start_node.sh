#!/bin/sh
# vim:sw=2:ts=2:et

set -e
[ "$LAVA_LOG_LEVEL" = "trace" ] && set -x

debug() {
	case "_${LAVA_LOG_LEVEL}" in
    _debug|_trace) echo "debug: $@" ;;
  esac
}

error() {
  echo "error: $@"
  exit 1
}

# defaults

lava_moniker_default='my-lava-testnet'
lava_chain_id_default='testnet-1'
lava_config_git_url_default='https://github.com/lavanet/lava-config.git'
lava_cosmovisor_url_default='https://lava-binary-upgrades.s3.amazonaws.com/testnet/cosmovisor-upgrades/cosmovisor-upgrades.zip'

# expect the following env vars:
#   LAVA_MONIKER        - lava node moniker (deafult: ${lava_moniker_default})
#   LAVA_CHAIN_ID       - lava chain identifier (default: ${lava_chain_id_default})
#   LAVA_CONFIG_GIT_URL - url to lavanet config assets (default: ${lava_config_git_url_default})
#   LAVA_COSMOVISOR_URL - url to lavanet cosmovisor zip (default: ${lava_cosmovisor_url_default})

LAVA_MONIKER=${LAVA_MONIKER:-${lava_moniker_default}}
LAVA_CHAIN_ID=${LAVA_CHAIN_ID:-${lava_chain_id_default}}
LAVA_COSMOVISOR_URL=${LAVA_COSMOVISOR_URL:-${lava_cosmovisor_url_default}}
LAVA_CONFIG_GIT_URL=${LAVA_CONFIG_GIT_URL:-${lava_config_git_url_default}}

# shortcuts
lava_home="$HOME/.lava"
lava_config="${lava_home}/config"
lava_binary="${lava_home}/cosmovisor/genesis/bin/lavad"

setup_env() {
  # environment variables for cosmovisor
  export CHAIN_ID=lava-${LAVA_CHAIN_ID}
  export DAEMON_NAME=lavad
  export DAEMON_HOME=${lava_home}
  export DAEMON_ALLOW_DOWNLOAD_BINARIES=true
  export DAEMON_LOG_BUFFER_SIZE=512
  export DAEMON_RESTART_AFTER_UPGRADE=true
  export UNSAFE_SKIP_BACKUP=true
}

setup_node() {
  local setup_config_dir

  debug "LAVA_CHAIN_ID=${LAVA_CHAIN_ID}"
  debug "LAVA_CONFIG_GIT_URL=${LAVA_CONFIG_GIT_URL}"
  debug "LAVA_COSMOVISOR_URL=${LAVA_COSMOVISOR_URL}"

  setup_config_dir=$(basename ${LAVA_CONFIG_GIT_URL})

  # remove old data (if any)
  rm -rf ${setup_config_dir}

  # download setup configuration
  git clone --depth 1 ${LAVA_CONFIG_GIT_URL} ${setup_config_dir} \
    || error "setup: failed to clone setup configuration"

  cd ${setup_config_dir}/${LAVA_CHAIN_ID} \
    || error "setup: invalid LAVA_CHAIN_ID '$LAVA_CHAIN_ID'"

  . setup_config/setup_config.sh \
    || error "setup: internal error (setup_config)"

  # keep a copy handy for when we restart
  cp -f setup_config/setup_config.sh ${HOME} \
    || error "setup: internal error (copy setup_config)"

  # remove old home/config (if any)
  rm -rf ${lava_home}
  rm -rf ${lava_config}

  # copy initial configuration and genesis data
  mkdir -p ${lava_home}
  mkdir -p ${lava_config}
  cp default_lavad_config_files/* ${lava_config}
  cp genesis_json/genesis.json ${lava_config}/genesis.json
}

# note: expected to run in the setup config directory; see setup_node()
setup_cosmovisor() {
  local output

  # download latest cosmovisor-upgrades
  curl -L --progress-bar -o cosmovisor-upgrades.zip "${LAVA_COSMOVISOR_URL}" \
    || error "setup: failed to download cosmovisor upgrades"
  unzip cosmovisor-upgrades.zip \
    || error "setup: failed to unzip cosmovisor upgrades"

  # copy cosmovisor configuration
  rm -rf ${lava_home}/cosmovisor/
  mv cosmovisor-upgrades/ ${lava_home}/cosmovisor/ \
    || error "setup: internal error (move cosmovisor)"

  # initialize the chain
  output=$( \
    ${lava_binary} init ${LAVA_MONIKER} \
      --chain-id ${LAVA_CHAIN_ID} \
      --home ${lavad_home} \
      --overwrite \
    )

  # an error message about missing upgrade-info.json is expected;
  # anything else is unexpected and should abort.
  if [ $? -ne 0 ]; then
    case "$output" in
    "*upgrade-info.json: no such file or directory*") ;;
    "*") error "setup: failed to initialize the chain" ;;
    esac
  fi

  # copy genesis data again
  cp genesis_json/genesis.json ${lava_config}/genesis.json \
    || error "setup: internal error (copy genesis.json)"
}

setup_env

if [ ! -e ${lava_home}/cosmovisor/current ]; then
  setup_node
  setup_cosmovisor
else
  . ${HOME}/setup_config.sh
fi

exec /bin/cosmovisor start --home=${lava_home} --p2p.seeds ${seed_node}

