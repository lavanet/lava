### FIRST MAKE SURE YOU BUILD LAVA DOCKER
# $ docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lavaC lava_starport |& grep -e lava_ -e ERR_ -e STARPORT] -e !


### RUN LAVA E2E TESTS - Runs all Lava e2e Tests - Full Lava Node, Providers, Test_clients
echo ' ::: LAVA E2E TEST DOCKER (STARPORT) :::'
docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lavaC lava_starport go test ./testutil/e2e/ -v 


### OR MANUALLY

### RUN LAVA DOCKER 
# echo ' ::: LAVA DOCKER (STARPORT) :::'
# docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lava19 lava_starport |& grep -e lava_ -e ERR_ -e STARPORT] -e ! &

# ### INIT
# echo ' ::: LAVA INIT (on docker):::'
# echo ' ::: waiting 40 secs :::'
# sleep 40
# docker exec -it lava19 sh .scripts/init.sh

# ### PROVIDERS
# echo ' ::: LAVA PROVIDERS (on docker):::'
# docker exec -it -d lava19 sh .scripts/eth.sh
# docker exec -it -d lava19 sh .scripts/osmosis.sh

# ### CLIENTS
# echo ' ::: LAVA CLIENT (on docker):::'
# sleep 15
# docker exec -it lava19 sh .scripts/run_clients.sh

