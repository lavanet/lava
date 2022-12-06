template = """{
    "name": "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
    "category": {
      "deterministic": true,
      "local": false,
      "subscription": false,
      "stateful": 0
    },
    "block_parsing": {
      "parserArg": [
      "latest"
      ],
      "parserFunc": "DEFAULT"
    },
    "compute_units": "10",
    "enabled": true,
    "apiInterfaces": [
      {
        "interface": "rest",
        "type": "get",
        "extra_compute_units": "0"
      }
    ]
  },"""
all_apis = ["/cosmwasm/wasm/v1/codes/pinned",
"/cosmwasm/wasm/v1/contract/{address}/raw/{query_data}",
"/cosmwasm/wasm/v1/contract/{address}/smart/{query_data}",
"/ibc/apps/interchain_accounts/host/v1/params",
"/ibc/apps/transfer/v1/denom_hashes/{trace}",
"/ibc/apps/transfer/v1/channels/{channel_id}/ports/{port_id}/escrow_address",
"/ibc/core/client/v1/client_status/{client_id}",
"/ibc/core/client/v1/consensus_states/{client_id}/heights",
"/ibc/core/client/v1/upgraded_client_states",
"/ibc/core/client/v1/upgraded_consensus_states",
"/osmosis/gamm/v1beta1/pool_type/{pool_id}",
"/osmosis/gamm/v1beta1/pools/{pool_id}/total_pool_liquidity",
"/osmosis/gamm/v1beta1/pools/{pool_id}/total_shares",
"/osmosis/incentives/v1beta1/active_gauges_per_denom",
"/osmosis/incentives/v1beta1/lockable_durations",
"/osmosis/incentives/v1beta1/upcoming_gauges_per_denom",
"/osmosis/lockup/v1beta1/account_locked_duration/{owner}",
"/osmosis/lockup/v1beta1/locked_denom",
"/osmosis/lockup/v1beta1/synthetic_lockups_by_lock_id/{lock_id}",
"/osmosis/pool-incentives/v1beta1/external_incentive_gauges",
"/osmosis/superfluid/v1beta1/all_intermediary_accounts",
"/osmosis/superfluid/v1beta1/asset_multiplier",
"/osmosis/superfluid/v1beta1/asset_type",
"/osmosis/superfluid/v1beta1/connected_intermediary_account/{lock_id}",
"/osmosis/superfluid/v1beta1/estimate_superfluid_delegation_amount_by_validator_denom",
"/osmosis/superfluid/v1beta1/params",
"/osmosis/superfluid/v1beta1/superfluid_delegation_amount",
"/osmosis/superfluid/v1beta1/superfluid_delegations/{delegator_address}",
"/osmosis/superfluid/v1beta1/superfluid_delegations_by_validator_denom",
"/osmosis/superfluid/v1beta1/superfluid_undelegations_by_delegator/{delegator_address}",
"/osmosis/superfluid/v1beta1/total_delegation_by_delegator/{delegator_address}",
"/osmosis/superfluid/v1beta1/all_superfluid_delegations",
"/osmosis/tokenfactory/v1beta1/denoms/{denom}/authority_metadata",
"/osmosis/tokenfactory/v1beta1/denoms_from_creator/{creator}",
"/osmosis/tokenfactory/v1beta1/params",
"/osmosis/twap/v1beta1/ArithmeticTwap",
"/osmosis/twap/v1beta1/ArithmeticTwapToNow",
"/osmosis/twap/v1beta1/Params",
"/osmosis/txfees/v1beta1/base_denom",
"/osmosis/txfees/v1beta1/denom_pool_id/{denom}",
"/osmosis/txfees/v1beta1/spot_price_by_denom",
"/osmosis/txfees/v1beta1/fee_tokens",]

final = ""
for api in all_apis:
    final += template.replace("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX", api)


with open("temp.json", "w+") as temp:
    temp.write(final)