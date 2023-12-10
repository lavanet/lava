# this script is adding another validator to the chain (without a running node) (this validator will be soon jailed due to inactivity)
clear
rm -rf ~/.lava_test
lavad init validator2 --chain-id lava --home ~/.lava_test
lavad config broadcast-mode sync --home ~/.lava_test
lavad config keyring-backend test --home ~/.lava_test
lavad keys add validator2 --home ~/.lava_test

GASPRICE="0.000000001ulava"
lavad tx bank send $(lavad keys show alice -a) $(lavad keys show validator2 -a --home ~/.lava_test) 500000ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE 

lavad tx staking create-validator -y --from validator2 --amount="50000ulava" --pubkey=$(lavad tendermint show-validator --home ~/.lava_test)  --commission-rate="0.10" \
    --commission-max-rate="0.20" \
    --commission-max-change-rate="0.01" \
	--min-self-delegation="1000" \
	--gas-adjustment "1.5" \
	--gas "auto" \
	--gas-prices $GASPRICE \
	--home ~/.lava_test
	
lavad tx staking redelegate lava@valoper1yhzkfrcdwf2hwpc4cre8er5tamp6wdm4stx2ec lava@valoper1z025w20ms6cpdht585nhsw682jph4yc7hx0gqc 500000000000ulava -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
