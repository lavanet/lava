##############################
#######   INIT LAVA    #######
##############################

### Init chain commands
sh scripts/init.sh 

### Run Providers (with mock proxies)
sh scripts/eth.sh &
sh scripts/osmosis.sh &

# For debug peroses - if you see stopwatch this providers are still running
# stopwatch &

#[+]