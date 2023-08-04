#!/bin/bash

# This is a script used for development and testing locally.
# Make sure you dont push the changes after this script ran.

echo "Buying subscription for user1"
GASPRICE="0.000000001ulava"
lavad tx subscription buy "DefaultPlan" -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

privateKey=$(yes | lavad keys export user1 --unsafe --unarmored-hex)
echo ""
echo "User1 Private Key: $privateKey"



# Capture unique iPPORT addresses and corresponding addresses
pairing=$(lavad q pairing providers LAV1)
addresses=$(echo "$pairing" | grep 'iPPORT:' | awk '{print $2}' | sort -u)
public_addresses=$(echo "$pairing" | grep 'address:' | awk '{print $3}')

# Associative array to store the JSON structure
declare -A json_data
index=0;
addresses_length=0;
for rpc_address in $addresses; do
    ((addresses_length++))
done
echo "Creating Pairing List json file"
for rpc_address in $addresses; do
    index_inner=0;
    for public_address in $public_addresses; do
        ((index_inner++))
        # echo "$index_inner,$addresses_length"
        if ((index_inner < index+1)); then
            continue
        fi
        if ((index_inner > index+1)); then
            break
        fi
        echo "Adding Provider: $rpc_address, $public_address"
        json_data["1"]+="{
        \"rpcAddress\": \"$rpc_address\",
        \"publicAddress\": \"$public_address\"
        }"
        if ((index < addresses_length-1)); then
            json_data["1"]+=","
        fi
    done
    ((index++))
done

# Construct the final JSON content
json_content='{
  "testnet": {
    "1": [
'

for index in "${!json_data[@]}"; do
    json_content+="  ${json_data["$index"]}"
    if ((index < ${#json_data[@]})); then
        json_content+=","
    fi
done

json_content+='  ]
  }
}'

# Write the JSON content to a file named "output.json"
echo "$json_content" > pairingList.json

# Run the badge server
signer=$(lavad keys show user1 -a)
echo "signer address: $signer"
PROJECT_ID="sampleProjectId"
GEOLOCATION="1"

cp examples/jsonRPC_badge.ts examples/jsonRPC_badge_test.ts

sed -i 's/geolocation: "2",/geolocation: "1",\n\n    pairingListConfig: "pairingList.json",\n\n    lavaChainId: "lava",/g' examples/jsonRPC_badge_test.ts

sed -i 's|badgeServerAddress:.*|badgeServerAddress: "http://localhost:8080",|g' examples/jsonRPC_badge_test.ts
sed -i "s|projectId:.*|projectId: \"$PROJECT_ID\",|g" examples/jsonRPC_badge_test.ts


BADGE_USER_DATA="{\"$GEOLOCATION\":{\"$PROJECT_ID\":{\"project_public_key\":\"$signer\",\"private_key\":\"$privateKey\",\"epochs_max_cu\":2233333333}}}" lavad badgegenerator --grpc-url=127.0.0.1:9090 --log_level=debug --chain-id lava

badgeResponse=$(curl -s -X POST -H "Content-Type: application/json" -d "{\"badge_address\": \"user1\", \"project_id\": \"$PROJECT_ID\"}" http://localhost:8080/lavanet.lava.pairing.BadgeGenerator/GenerateBadge)

if [[ -z "$badgeResponse" ]]; then
  echo "Failed to generate the badge. Please check if the badge server is running and accessible."
  exit 1
fi

echo "$badgeResponse"

echo "Done."
