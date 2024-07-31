#!/bin/bash

# This is a script used for development and testing locally.
# Make sure you dont push the changes after this script ran.

echo "Buying subscription for user1"
GASPRICE="0.00002ulava"
lavad tx subscription buy "DefaultPlan" -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

privateKey=$(yes | lavad keys export user1 --unsafe --unarmored-hex)
echo ""
echo "User1 Private Key: $privateKey"



# Capture unique iPPORT addresses and corresponding addresses
pairing=$(lavad q pairing providers LAV1)
addresses=$(echo "$pairing" | grep 'iPPORT:' | awk '{print $2}' | sort -u)
public_addresses=$(echo "$pairing" | grep 'address:' | awk '{print $3}')

GEOLOCATION=2

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
        json_data["$GEOLOCATION"]+="{
        \"rpcAddress\": \"$rpc_address\",
        \"publicAddress\": \"$public_address\"
        }"
        if ((index < addresses_length-1)); then
            json_data["$GEOLOCATION"]+=","
        fi
    done
    ((index++))
done

# Construct the final JSON content
json_content="{
  \"testnet\": {
    \"$GEOLOCATION\": [
"

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
echo "Signer address: $signer"
PROJECT_ID="sampleProjectId"
BADGE_PORT=8081
BADGE_URL=http://localhost:$BADGE_PORT

cp examples/jsonRPC_badge.ts examples/jsonRPC_badge_test.ts
cp examples/restAPI_badge.ts examples/restAPI_badge_test.ts
cp examples/tendermintRPC_badge.ts examples/tendermintRPC_badge_test.ts

sed -i "s|geolocation:.*,|geolocation: \"$GEOLOCATION\",\n\n    pairingListConfig: \"pairingList.json\",\n\n    lavaChainId: \"lava\",\n\n    logLevel: \"debug\",\n\n    allowInsecureTransport: true|g" examples/jsonRPC_badge_test.ts
sed -i "s|geolocation:.*,|geolocation: \"$GEOLOCATION\",\n\n    pairingListConfig: \"pairingList.json\",\n\n    lavaChainId: \"lava\",\n\n    logLevel: \"debug\",\n\n    allowInsecureTransport: true|g" examples/restAPI_badge_test.ts
sed -i "s|geolocation:.*,|geolocation: \"$GEOLOCATION\",\n\n    pairingListConfig: \"pairingList.json\",\n\n    lavaChainId: \"lava\",\n\n    logLevel: \"debug\",\n\n    allowInsecureTransport: true|g" examples/tendermintRPC_badge_test.ts

sed -i "s|badgeServerAddress:.*|badgeServerAddress: \"$BADGE_URL\",|g" examples/jsonRPC_badge_test.ts
sed -i "s|badgeServerAddress:.*|badgeServerAddress: \"$BADGE_URL\",|g" examples/restAPI_badge_test.ts
sed -i "s|badgeServerAddress:.*|badgeServerAddress: \"$BADGE_URL\",|g" examples/tendermintRPC_badge_test.ts

sed -i "s|projectId:.*|projectId: \"$PROJECT_ID\",|g" examples/jsonRPC_badge_test.ts
sed -i "s|projectId:.*|projectId: \"$PROJECT_ID\",|g" examples/restAPI_badge_test.ts
sed -i "s|projectId:.*|projectId: \"$PROJECT_ID\",|g" examples/tendermintRPC_badge_test.ts

badgeserverconfigdirectory=~/go/lava/config
badgeserverconfigfile=$badgeserverconfigdirectory/badgeserver.yml
sed -i '/^projects-data:/,/^  [0-9]*:/ s/^  [0-9]*:/'"  $GEOLOCATION"':/' $badgeserverconfigfile
sed -i 's/^default-geolocation:.*/default-geolocation: '$GEOLOCATION'/' $badgeserverconfigfile
sed -i '/^    default:/ s/default:/'$PROJECT_ID':/' $badgeserverconfigfile

lavap badgeserver $badgeserverconfigdirectory --log_level=debug --chain-id lava --port $BADGE_PORT --from user1

badgeResponse=$(curl -s -X POST -H "Content-Type: application/json" -d "{\"badge_address\": \"$signer\", \"project_id\": \"$PROJECT_ID\"}" $BADGE_URL/lavanet.lava.pairing.BadgeGenerator/GenerateBadge)

if [[ -z "$badgeResponse" ]]; then
  echo "Failed to generate the badge. Please check if the badge server is running and accessible."
  exit 1
fi

echo "$badgeResponse"

echo "Done."
