##############################
######   RUN CLIENTS    ######
##############################

### Init chain commands
# ETH clients
3; go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client ETH1 jsonrpc --from user1&
# Osmosis clients
3; go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client COS3 tendermintrpc --from user2
# Osmosis clients again to generate relay payment
sleep 40
3; go run /home/magic/go/lava/relayer/cmd/relayer/main.go test_client COS3 tendermintrpc --from user2

# [+]