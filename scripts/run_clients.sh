##############################
######   RUN CLIENTS    ######
##############################

### Init chain commands
# ETH clients
3; lavad test_client ETH1 jsonrpc --from user1&
# Osmosis clients
3; lavad test_client COS3 tendermintrpc --from user2
# Osmosis clients again to generate relay payment
sleep 40
3; lavad test_client COS3 tendermintrpc --from user2

# [+]