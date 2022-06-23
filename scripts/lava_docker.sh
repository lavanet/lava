### FIRST MAKE SURE YOU BUILD LAVA DOCKER
# $ docker build . -t lava_ignite


### RUN LAVA E2E TESTS - Runs all Lava e2e Tests - Full Lava Node, Providers, Test_clients
echo ' ::: LAVA E2E TEST DOCKER (IGNITE) :::'
docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lavaC lava_ignite go test ./testutil/e2e/ -v 

### OR MANUALLY

### RUN LAVA DOCKER 
echo ' ::: LAVA DOCKER (STARPORT) :::'
docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 --name lavaC lava_starport |& grep -e lava_ -e ERR_ -e STARPORT] -e ! 

# ### INIT
# echo ' ::: LAVA INIT (on docker):::'
# echo ' ::: waiting 40 secs :::'
# sleep 40
# docker exec -it lavaC sh scripts/init.sh

# ### PROVIDERS
# echo ' ::: LAVA PROVIDERS (on docker):::'
# docker exec -it -d lavaC sh scripts/eth.sh
# docker exec -it -d lavaC sh scripts/osmosis.sh

# ### CLIENTS
# echo ' ::: LAVA CLIENT (on docker):::'
# sleep 15
# docker exec -it lavaC sh scripts/run_clients.sh



### EXPERIMENTAL
# docker run -p 4500:4500 -p 1317:1317 -p 26657:26657  -v $LAVA/docker/shared:/go/lava/docker/shared --name lavaC lava_starport ignite chain serve -v --home /go/lava/docker/shared |& grep -e lava_ -e ERR_ -e STARPORT] -e ! 
# docker run -p 4500:4500 -p 1317:1317 -p 26657:26657 \
#   --mount 'type=volume,src=/home/magic/go/lava/docker/shared,dst=/go/lava/docker/shared,volume-driver=local,readonly' \ 
#   lava_starport ignite chain serve -v --home /go/lava/docker/shared |& grep -e lava_ -e ERR_ -e STARPORT] -e ! 

