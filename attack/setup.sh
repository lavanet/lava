#!/bin/bash
set -e

BASE_URL="https://base-jsonrpc.nadav.magmadevs.com:443"
ETH_URL="https://eth-jsonrpc.nadav.magmadevs.com:443"
SOL_URL="https://solana-jsonrpc.nadav.magmadevs.com:443"

mkdir -p bodies results

# ─── Light ────────────────────────────────────────────────────────────────────
cat > bodies/eth_blockNumber.json       <<'EOF'
{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}
EOF
cat > bodies/eth_chainId.json           <<'EOF'
{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}
EOF
cat > bodies/eth_gasPrice.json          <<'EOF'
{"jsonrpc":"2.0","method":"eth_gasPrice","params":[],"id":1}
EOF
cat > bodies/eth_getBlockByNumber.json  <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",false],"id":1}
EOF
cat > bodies/eth_getBlockByNumber_full.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",true],"id":1}
EOF

# ─── Archive ──────────────────────────────────────────────────────────────────
# eth_getBalance at historical block (vitalik.eth at block 14 000 000)
cat > bodies/eth_getBalance_archive_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBalance","params":["0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045","0xD59F80"],"id":1}
EOF
# Base: WETH contract at block 5 000 000
cat > bodies/eth_getBalance_archive_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x4200000000000000000000000000000000000006","0x4C4B40"],"id":1}
EOF
# eth_getCode at historical block
cat > bodies/eth_getCode_archive_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getCode","params":["0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","0xD59F80"],"id":1}
EOF
cat > bodies/eth_getCode_archive_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getCode","params":["0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","0x4C4B40"],"id":1}
EOF
# eth_call: USDC balanceOf(vitalik) at historical block
cat > bodies/eth_call_archive_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","data":"0x70a08231000000000000000000000000d8dA6BF26964aF9D7eEd9e03E53415D37aA96045"},"0xD59F80"],"id":1}
EOF
cat > bodies/eth_call_archive_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","data":"0x70a08231000000000000000000000000d8dA6BF26964aF9D7eEd9e03E53415D37aA96045"},"0x4C4B40"],"id":1}
EOF

# ─── EVM Light extras ─────────────────────────────────────────────────────────
cat > bodies/eth_syncing.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}
EOF
cat > bodies/net_version.json <<'EOF'
{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}
EOF
cat > bodies/web3_clientVersion.json <<'EOF'
{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}
EOF
cat > bodies/net_listening.json <<'EOF'
{"jsonrpc":"2.0","method":"net_listening","params":[],"id":1}
EOF
cat > bodies/eth_protocolVersion.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_protocolVersion","params":[],"id":1}
EOF
cat > bodies/eth_accounts.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}
EOF

# ─── EVM Medium ──────────────────────────────────────────────────────────────
cat > bodies/eth_feeHistory.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_feeHistory","params":["0x4","latest",[25,75]],"id":1}
EOF
cat > bodies/eth_maxPriorityFeePerGas.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_maxPriorityFeePerGas","params":[],"id":1}
EOF
# Base: nonce of USDC contract
cat > bodies/eth_getTransactionCount_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","latest"],"id":1}
EOF
# ETH: nonce of USDC contract
cat > bodies/eth_getTransactionCount_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getTransactionCount","params":["0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","latest"],"id":1}
EOF
# Base: recent logs from USDC
cat > bodies/eth_getLogs_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"address":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","fromBlock":"latest","toBlock":"latest"}],"id":1}
EOF
# ETH: recent logs from USDC
cat > bodies/eth_getLogs_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"address":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","fromBlock":"latest","toBlock":"latest"}],"id":1}
EOF
# Storage slot 0 of USDC
cat > bodies/eth_getStorageAt_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","0x0","latest"],"id":1}
EOF
cat > bodies/eth_getStorageAt_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getStorageAt","params":["0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","0x0","latest"],"id":1}
EOF
# Estimate gas for a simple transfer
cat > bodies/eth_estimateGas_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{"from":"0x0000000000000000000000000000000000000000","to":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","value":"0x0"}],"id":1}
EOF
cat > bodies/eth_estimateGas_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_estimateGas","params":[{"from":"0x0000000000000000000000000000000000000000","to":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","value":"0x0"}],"id":1}
EOF
# Block transaction count
cat > bodies/eth_getBlockTransactionCountByNumber.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBlockTransactionCountByNumber","params":["latest"],"id":1}
EOF
# eth_call at latest (totalSupply on USDC)
cat > bodies/eth_call_latest_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","data":"0x18160ddd"},"latest"],"id":1}
EOF
cat > bodies/eth_call_latest_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","data":"0x18160ddd"},"latest"],"id":1}
EOF
# createAccessList
cat > bodies/eth_createAccessList_base.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_createAccessList","params":[{"from":"0x0000000000000000000000000000000000000000","to":"0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913","data":"0x18160ddd"}],"id":1}
EOF
cat > bodies/eth_createAccessList_eth.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_createAccessList","params":[{"from":"0x0000000000000000000000000000000000000000","to":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","data":"0x18160ddd"}],"id":1}
EOF

# ─── EVM Heavy ───────────────────────────────────────────────────────────────
# Block receipts for latest block
cat > bodies/eth_getBlockReceipts.json <<'EOF'
{"jsonrpc":"2.0","method":"eth_getBlockReceipts","params":["latest"],"id":1}
EOF

# ─── Debug ────────────────────────────────────────────────────────────────────
cat > bodies/debug_traceBlockByNumber_call.json <<'EOF'
{"jsonrpc":"2.0","method":"debug_traceBlockByNumber","params":["latest",{"tracer":"callTracer"}],"id":1}
EOF
cat > bodies/debug_traceBlockByNumber_prestate.json <<'EOF'
{"jsonrpc":"2.0","method":"debug_traceBlockByNumber","params":["latest",{"tracer":"prestateTracer"}],"id":1}
EOF

# ─── Trace ────────────────────────────────────────────────────────────────────
cat > bodies/trace_block.json <<'EOF'
{"jsonrpc":"2.0","method":"trace_block","params":["latest"],"id":1}
EOF
cat > bodies/trace_replayBlockTransactions.json <<'EOF'
{"jsonrpc":"2.0","method":"trace_replayBlockTransactions","params":["latest",["trace"]],"id":1}
EOF
cat > bodies/trace_replayBlockTransactions_vm.json <<'EOF'
{"jsonrpc":"2.0","method":"trace_replayBlockTransactions","params":["latest",["trace","vmTrace","stateDiff"]],"id":1}
EOF

# ─── Solana ───────────────────────────────────────────────────────────────────
# Light
cat > bodies/sol_getSlot.json <<'EOF'
{"jsonrpc":"2.0","method":"getSlot","id":1}
EOF
cat > bodies/sol_getBlockHeight.json <<'EOF'
{"jsonrpc":"2.0","method":"getBlockHeight","id":1}
EOF
cat > bodies/sol_getHealth.json <<'EOF'
{"jsonrpc":"2.0","method":"getHealth","id":1}
EOF
cat > bodies/sol_getVersion.json <<'EOF'
{"jsonrpc":"2.0","method":"getVersion","id":1}
EOF
cat > bodies/sol_getLatestBlockhash.json <<'EOF'
{"jsonrpc":"2.0","method":"getLatestBlockhash","params":[{"commitment":"finalized"}],"id":1}
EOF
cat > bodies/sol_getEpochInfo.json <<'EOF'
{"jsonrpc":"2.0","method":"getEpochInfo","id":1}
EOF
cat > bodies/sol_getRecentPerformanceSamples.json <<'EOF'
{"jsonrpc":"2.0","method":"getRecentPerformanceSamples","params":[5],"id":1}
EOF

# Medium — account reads
# SOL balance of a known account (Toly's wallet)
cat > bodies/sol_getBalance.json <<'EOF'
{"jsonrpc":"2.0","method":"getBalance","params":["CRFZscVLBpHswKRPwrd7XNdnqJsCXaq3gFMXTaUspm3k"],"id":1}
EOF
cat > bodies/sol_getAccountInfo.json <<'EOF'
{"jsonrpc":"2.0","method":"getAccountInfo","params":["TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",{"encoding":"base64"}],"id":1}
EOF
cat > bodies/sol_getTokenAccountsByOwner.json <<'EOF'
{"jsonrpc":"2.0","method":"getTokenAccountsByOwner","params":["CRFZscVLBpHswKRPwrd7XNdnqJsCXaq3gFMXTaUspm3k",{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"},{"encoding":"jsonParsed"}],"id":1}
EOF
cat > bodies/sol_getSignaturesForAddress.json <<'EOF'
{"jsonrpc":"2.0","method":"getSignaturesForAddress","params":["CRFZscVLBpHswKRPwrd7XNdnqJsCXaq3gFMXTaUspm3k",{"limit":5}],"id":1}
EOF

# Medium extras
cat > bodies/sol_getSupply.json <<'EOF'
{"jsonrpc":"2.0","method":"getSupply","id":1}
EOF
cat > bodies/sol_getInflationRate.json <<'EOF'
{"jsonrpc":"2.0","method":"getInflationRate","id":1}
EOF
cat > bodies/sol_getMinimumBalanceForRentExemption.json <<'EOF'
{"jsonrpc":"2.0","method":"getMinimumBalanceForRentExemption","params":[165],"id":1}
EOF
cat > bodies/sol_getGenesisHash.json <<'EOF'
{"jsonrpc":"2.0","method":"getGenesisHash","id":1}
EOF
cat > bodies/sol_getIdentity.json <<'EOF'
{"jsonrpc":"2.0","method":"getIdentity","id":1}
EOF
cat > bodies/sol_getFirstAvailableBlock.json <<'EOF'
{"jsonrpc":"2.0","method":"getFirstAvailableBlock","id":1}
EOF
cat > bodies/sol_getStakeMinimumDelegation.json <<'EOF'
{"jsonrpc":"2.0","method":"getStakeMinimumDelegation","id":1}
EOF
cat > bodies/sol_getRecentPrioritizationFees.json <<'EOF'
{"jsonrpc":"2.0","method":"getRecentPrioritizationFees","id":1}
EOF
cat > bodies/sol_getTokenSupply.json <<'EOF'
{"jsonrpc":"2.0","method":"getTokenSupply","params":["EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"],"id":1}
EOF
cat > bodies/sol_getMultipleAccounts.json <<'EOF'
{"jsonrpc":"2.0","method":"getMultipleAccounts","params":[["TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","11111111111111111111111111111111"],{"encoding":"base64"}],"id":1}
EOF
cat > bodies/sol_isBlockhashValid.json <<'EOF'
{"jsonrpc":"2.0","method":"isBlockhashValid","params":["J7rBdM6AecPDEZp8aPq5iPSNKVkU5Q76F3oAV4eW5wsW",{"commitment":"processed"}],"id":1}
EOF

# Heavy
cat > bodies/sol_getClusterNodes.json <<'EOF'
{"jsonrpc":"2.0","method":"getClusterNodes","id":1}
EOF
cat > bodies/sol_getLargestAccounts.json <<'EOF'
{"jsonrpc":"2.0","method":"getLargestAccounts","id":1}
EOF
cat > bodies/sol_getLeaderSchedule.json <<'EOF'
{"jsonrpc":"2.0","method":"getLeaderSchedule","id":1}
EOF

# Heavy (existing)
cat > bodies/sol_getProgramAccounts.json <<'EOF'
{"jsonrpc":"2.0","method":"getProgramAccounts","params":["TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",{"encoding":"base64","dataSlice":{"offset":0,"length":0},"filters":[{"dataSize":165},{"memcmp":{"offset":32,"bytes":"CRFZscVLBpHswKRPwrd7XNdnqJsCXaq3gFMXTaUspm3k"}}]}],"id":1}
EOF
cat > bodies/sol_getBlockProduction.json <<'EOF'
{"jsonrpc":"2.0","method":"getBlockProduction","id":1}
EOF
cat > bodies/sol_getVoteAccounts.json <<'EOF'
{"jsonrpc":"2.0","method":"getVoteAccounts","id":1}
EOF

# ─── Targets ──────────────────────────────────────────────────────────────────
# Base: trace_block and trace_replayBlockTransactions return -32601 (not supported by Base nodes)
build_targets_base() {
  local url=$BASE_URL
  cat <<EOF
POST ${url}
Content-Type: application/json
@bodies/eth_blockNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_chainId.json

POST ${url}
Content-Type: application/json
@bodies/eth_gasPrice.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockByNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockByNumber_full.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBalance_archive_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_getCode_archive_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_call_archive_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_syncing.json

POST ${url}
Content-Type: application/json
@bodies/net_version.json

POST ${url}
Content-Type: application/json
@bodies/web3_clientVersion.json

POST ${url}
Content-Type: application/json
@bodies/net_listening.json

POST ${url}
Content-Type: application/json
@bodies/eth_protocolVersion.json

POST ${url}
Content-Type: application/json
@bodies/eth_accounts.json

POST ${url}
Content-Type: application/json
@bodies/eth_feeHistory.json

POST ${url}
Content-Type: application/json
@bodies/eth_maxPriorityFeePerGas.json

POST ${url}
Content-Type: application/json
@bodies/eth_getTransactionCount_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_getLogs_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_getStorageAt_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_estimateGas_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockTransactionCountByNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_call_latest_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_createAccessList_base.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockReceipts.json

POST ${url}
Content-Type: application/json
@bodies/debug_traceBlockByNumber_call.json

POST ${url}
Content-Type: application/json
@bodies/debug_traceBlockByNumber_prestate.json
EOF
}

build_targets_eth() {
  local url=$ETH_URL
  cat <<EOF
POST ${url}
Content-Type: application/json
@bodies/eth_blockNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_chainId.json

POST ${url}
Content-Type: application/json
@bodies/eth_gasPrice.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockByNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockByNumber_full.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBalance_archive_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_getCode_archive_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_call_archive_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_syncing.json

POST ${url}
Content-Type: application/json
@bodies/net_version.json

POST ${url}
Content-Type: application/json
@bodies/web3_clientVersion.json

POST ${url}
Content-Type: application/json
@bodies/net_listening.json

POST ${url}
Content-Type: application/json
@bodies/eth_protocolVersion.json

POST ${url}
Content-Type: application/json
@bodies/eth_accounts.json

POST ${url}
Content-Type: application/json
@bodies/eth_feeHistory.json

POST ${url}
Content-Type: application/json
@bodies/eth_maxPriorityFeePerGas.json

POST ${url}
Content-Type: application/json
@bodies/eth_getTransactionCount_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_getLogs_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_getStorageAt_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_estimateGas_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockTransactionCountByNumber.json

POST ${url}
Content-Type: application/json
@bodies/eth_call_latest_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_createAccessList_eth.json

POST ${url}
Content-Type: application/json
@bodies/eth_getBlockReceipts.json

POST ${url}
Content-Type: application/json
@bodies/debug_traceBlockByNumber_call.json

POST ${url}
Content-Type: application/json
@bodies/debug_traceBlockByNumber_prestate.json

POST ${url}
Content-Type: application/json
@bodies/trace_block.json

POST ${url}
Content-Type: application/json
@bodies/trace_replayBlockTransactions.json

POST ${url}
Content-Type: application/json
@bodies/trace_replayBlockTransactions_vm.json
EOF
}

build_targets_sol() {
  local url=$SOL_URL
  cat <<EOF
POST ${url}
Content-Type: application/json
@bodies/sol_getSlot.json

POST ${url}
Content-Type: application/json
@bodies/sol_getBlockHeight.json

POST ${url}
Content-Type: application/json
@bodies/sol_getHealth.json

POST ${url}
Content-Type: application/json
@bodies/sol_getVersion.json

POST ${url}
Content-Type: application/json
@bodies/sol_getLatestBlockhash.json

POST ${url}
Content-Type: application/json
@bodies/sol_getEpochInfo.json

POST ${url}
Content-Type: application/json
@bodies/sol_getRecentPerformanceSamples.json

POST ${url}
Content-Type: application/json
@bodies/sol_getBalance.json

POST ${url}
Content-Type: application/json
@bodies/sol_getAccountInfo.json

POST ${url}
Content-Type: application/json
@bodies/sol_getTokenAccountsByOwner.json

POST ${url}
Content-Type: application/json
@bodies/sol_getSignaturesForAddress.json

POST ${url}
Content-Type: application/json
@bodies/sol_getSupply.json

POST ${url}
Content-Type: application/json
@bodies/sol_getInflationRate.json

POST ${url}
Content-Type: application/json
@bodies/sol_getMinimumBalanceForRentExemption.json

POST ${url}
Content-Type: application/json
@bodies/sol_getGenesisHash.json

POST ${url}
Content-Type: application/json
@bodies/sol_getIdentity.json

POST ${url}
Content-Type: application/json
@bodies/sol_getFirstAvailableBlock.json

POST ${url}
Content-Type: application/json
@bodies/sol_getStakeMinimumDelegation.json

POST ${url}
Content-Type: application/json
@bodies/sol_getRecentPrioritizationFees.json

POST ${url}
Content-Type: application/json
@bodies/sol_getTokenSupply.json

POST ${url}
Content-Type: application/json
@bodies/sol_getMultipleAccounts.json

POST ${url}
Content-Type: application/json
@bodies/sol_isBlockhashValid.json

POST ${url}
Content-Type: application/json
@bodies/sol_getProgramAccounts.json

POST ${url}
Content-Type: application/json
@bodies/sol_getBlockProduction.json

POST ${url}
Content-Type: application/json
@bodies/sol_getVoteAccounts.json

POST ${url}
Content-Type: application/json
@bodies/sol_getClusterNodes.json

POST ${url}
Content-Type: application/json
@bodies/sol_getLargestAccounts.json

POST ${url}
Content-Type: application/json
@bodies/sol_getLeaderSchedule.json
EOF
}

build_targets_base > targets_base.txt
build_targets_eth  > targets_eth.txt
build_targets_sol  > targets_sol.txt

# Mixed targets (all endpoints concatenated)
cat targets_base.txt targets_eth.txt targets_sol.txt > targets_mixed.txt

echo "Setup complete. Bodies: $(ls bodies/*.json | wc -l | tr -d ' ')"
echo "Targets: targets_base.txt  targets_eth.txt  targets_sol.txt  targets_mixed.txt"
