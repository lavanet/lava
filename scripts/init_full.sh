##############################
#######   INIT LAVA    #######
##############################

### Init chain commands
sh scripts/init.sh 

### Run Providers (with mock proxies)
sh scripts/osmosis.sh &
sh scripts/eth.sh &
sh scripts/gth.sh &
sh scripts/ftm.sh &

# For debug peroses - if you see stopwatch this providers are still running
# stopwatch &

#[+]