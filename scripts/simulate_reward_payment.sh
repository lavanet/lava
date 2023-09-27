#!/bin/bash 

echo "Executing init_chain_commands"
./scripts/init_chain_commands.sh --skip-providers

# Step 1: Get the balance of servicer1
initial_balance=$(lavad q bank balances $(lavad keys show servicer1 -a))
echo "Initial Balance of servicer1: $initial_balance"
initial_value=$(echo "$initial_balance" | awk -F\" '/amount: / {print $2}')

# Step 2: Run simulate tx
lavad tx pairing simulate-relay-payment user1 LAV1 --from servicer1 --cu-amount 200 --gas-prices 0.00001ulava --gas-adjustment "1.5" --gas "auto" --relay-num 5 --yes
sleep 2

# Step 3: Check the balance again
final_balance=$(lavad q bank balances $(lavad keys show servicer1 -a))
echo "Final Balance of servicer1: $final_balance"
final_value=$(echo "$final_balance" | awk -F\" '/amount: / {print $2}')

if (( final_value > initial_value )); then
    echo "Success: Final balance is greater than the initial balance"
else
    echo "Error: Final balance is not greater than the initial balance"
    exit 1
fi