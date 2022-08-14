##############################
#######   INIT LAVA    #######
##############################

### Init chain commands
sh scripts/e2e/init.sh 

### Run Providers (with mock proxies)
sh scripts/e2e/eth.sh &
sh scripts/e2e/gth.sh &
sh scripts/e2e/ftm.sh &
sh scripts/e2e/osmosis.sh &

# For debug peroses - if you see stopwatch this providers are still running
# stopwatch &

#[+]