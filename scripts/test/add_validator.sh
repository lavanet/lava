# this script is adding another validator to the chain (without a running node) (this validator will be soon jailed due to inactivity)
clear
home=~/.lava2
rm -rf $home
lavad init validator2 --chain-id lava --home $home
lavad config broadcast-mode sync --home $home
lavad config keyring-backend test --home $home
lavad keys add validator2 --home $home

cp ~/.lava/config/addrbook.json $home/config/addrbook.json
cp ~/.lava/config/app.toml $home/config/app.toml
cp ~/.lava/config/client.toml $home/config/client.toml
cp ~/.lava/config/genesis.json $home/config/genesis.json


# Specify the file path, field to edit, and new value
path="$home/config/"
config='config.toml'
app='app.toml'

# Determine OS
os_name=$(uname)
case "$(uname)" in
  Darwin)
    SED_INLINE="-i ''" ;;
  Linux)
    SED_INLINE="-i" ;;
  *)
    echo "unknown system: $(uname)"
    exit 1 ;;
esac

sed $SED_INLINE \
-e 's/tcp:\/\/0\.0\.0\.0:26656/tcp:\/\/0.0.0.0:36656/' \
-e 's/tcp:\/\/127\.0\.0\.1:26658/tcp:\/\/127.0.0.1:36658/' \
-e 's/tcp:\/\/127\.0\.0\.1:26657/tcp:\/\/127.0.0.1:36657/' \
-e 's/tcp:\/\/127\.0\.0\.1:26656/tcp:\/\/127.0.0.1:36656/' "$path$config"

# Edit app.toml file
sed $SED_INLINE \
-e 's/tcp:\/\/localhost:1317/tcp:\/\/localhost:2317/' \
-e 's/localhost:9090/localhost:8090/' \
-e 's/":7070"/":7070"/' \
-e 's/localhost:9091/localhost:8091/' "$path$app"


GASPRICE="0.00002ulava"
lavad tx bank send $(lavad keys show alice -a) $(lavad keys show validator2 -a --home $home) 10000000000001ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE 
sleep 3

lavad tx staking create-validator -y --from validator2 --amount="10000000000000ulava" --pubkey=$(lavad tendermint show-validator --home $home) \
	--commission-rate="0.10" \
    --commission-max-rate="0.20" \
    --commission-max-change-rate="0.01" \
	--min-self-delegation="1000" \
	--gas-adjustment "1.5" \
	--gas "auto" \
	--gas-prices $GASPRICE \
	--home $home

id=$(lavad status | jq .NodeInfo.id -r)
addr=$(lavad status | jq .NodeInfo.listen_addr -r | sed 's/tcp:\/\///')
lavad start --home $home --p2p.seeds $id@$addr

	
