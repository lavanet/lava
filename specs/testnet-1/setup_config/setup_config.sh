# Configurations

# Packages
deploy_env="testnet-1"
# The network block time +1 second - used for sleep command between txs
block_time=61
dependency_packages="unzip logrotate git jq sed wget curl coreutils systemd"
go_package_url="https://go.dev/dl/go1.18.linux-amd64.tar.gz"
go_package_file_name=${go_package_url##*\/}
ignite_package="https://get.ignite.com/cli!"
lava_github_repo="git@github.com:lavanet/lava.git"
binary_url="https://lava-pnet0-setup.s3.amazonaws.com/release/$deploy_env/latest/lavad"

# Nginx configurations
# Generate a random password with the length of the 42 characters
nginx_username=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 42 | head -n 1)
# Generate a random username with the length of the 42 characters
secret_password=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 42 | head -n 1)

# Blockchain
seed_node="3a445bfdbe2d0c8ee82461633aa3af31bc2b4dc0@prod-pnet-seed-node.lavanet.xyz:26656,e593c7a9ca61f5616119d6beb5bd8ef5dd28d62d@prod-pnet-seed-node2.lavanet.xyz:26656"

# Lavad service
keyring_backend="test"
# 600 minutes = 36000 seconds, sleep time is 10 seconds so retry count is 3600
catch_up_retry_count=3600
default_config_files_url="https://lava-pnet0-setup.s3.amazonaws.com/config/default_lavad_config_files/lavad_config_$deploy_env.zip"
genesis_url="https://lava-pnet0-setup.s3.amazonaws.com/config/genesis_$deploy_env.json"
lava_config_folder="$HOME/.lava/config"
lavad_home_folder="$HOME/.lava"
validator_stake_amount="50000000ulava"
provider_stake_amount="2010ulava"
