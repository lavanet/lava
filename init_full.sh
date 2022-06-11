##############################
#######   INIT LAVA    #######
##############################

### Init chain commands
sh .scripts/init.sh 
### Run mock proxies
sh .scripts/mock_proxy_eth.sh &
sh .scripts/mock_proxy_osmosis.sh &

### Run Providers
sh .scripts/providers_eth_mock.sh &
sh .scripts/providers_osmosis_mock.sh &

# For debug peroses - if you see stopwatch this providers are still running
# stopwatch &

#[+]