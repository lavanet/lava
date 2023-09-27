package cli

import (
	"context"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/spf13/cobra"
)

const (
	AmountFlagName    = "amount"
	EndpointsFlagName = "endpoints"
	GeolocationFlag   = "geolocation"
)

var _ = strconv.Itoa(0)

func CmdModifyProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify-provider [chain-id] --from <address>",
		Short: `modify a staked provider on the lava blockchain on a specific specification, provider must be already staked`,
		Long: `args:
		[chain-id] is the spec the provider wishes to modify the entry for
		`,
		Example: `lavad tx pairing modify-provider "ETH1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainID := args[0]
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()
			keyName, err := sigs.GetKeyName(clientCtx)
			if err != nil {
				utils.LavaFormatFatal("failed getting key name from clientCtx", err)
			}
			clientKey, err := clientCtx.Keyring.Key(keyName)
			if err != nil {
				return err
			}
			address, err := clientKey.GetAddress()
			if err != nil {
				return err
			}

			pairingQuerier := types.NewQueryClient(clientCtx)
			response, err := pairingQuerier.Providers(ctx, &types.QueryProvidersRequest{
				ChainID:    argChainID,
				ShowFrozen: true,
			})
			if err != nil {
				return err
			}
			if len(response.StakeEntry) == 0 {
				return utils.LavaFormatError("provider isn't staked on chainID, no providers at all", nil)
			}
			var providerEntry *epochstoragetypes.StakeEntry
			for idx, provider := range response.StakeEntry {
				if provider.Address == address.String() {
					providerEntry = &response.StakeEntry[idx]
					break
				}
			}
			if providerEntry == nil {
				return utils.LavaFormatError("provider isn't staked on chainID, no address match", nil)
			}
			newAmount, err := cmd.Flags().GetString(AmountFlagName)
			if err != nil {
				return err
			}
			if newAmount != "" {
				newStake, err := sdk.ParseCoinNormalized(newAmount)
				if err != nil {
					return err
				}
				if providerEntry.Stake.Amount.GT(newStake.Amount) {
					return utils.LavaFormatError("can't reduce provider stake", nil, utils.Attribute{Key: "current", Value: providerEntry.Stake}, utils.Attribute{Key: "requested", Value: providerEntry.Stake})
				}
				providerEntry.Stake = newStake
			}
			geolocation, err := cmd.Flags().GetInt32(GeolocationFlag)
			if err != nil {
				return err
			}
			if geolocation != 0 {
				providerEntry.Geolocation = geolocation
			}
			newEndpointsStr, err := cmd.Flags().GetString(EndpointsFlagName)
			if err != nil {
				return err
			}
			if newEndpointsStr != "" {
				tmpArg := strings.Fields(newEndpointsStr)
				argEndpoints, _, err := HandleEndpointsAndGeolocationArgs(tmpArg, strconv.FormatInt(int64(providerEntry.Geolocation), 10))
				if err != nil {
					return err
				}
				providerEntry.Endpoints = argEndpoints
			}

			moniker, err := cmd.Flags().GetString(types.FlagMoniker)
			if err != nil {
				return err
			}
			if moniker != "" {
				providerEntry.Moniker = moniker
			}

			if cmd.Flags().Changed(types.FlagCommission) {
				providerEntry.DelegateCommission, err = cmd.Flags().GetUint64(types.FlagCommission)
				if err != nil {
					return err
				}
			}

			if cmd.Flags().Changed(types.FlagDelegationLimit) {
				delegationLimitStr, err := cmd.Flags().GetString(types.FlagDelegationLimit)
				if err != nil {
					return err
				}
				providerEntry.DelegateLimit, err = sdk.ParseCoinNormalized(delegationLimitStr)
				if err != nil {
					return err
				}
			}

			// modify fields
			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				argChainID,
				providerEntry.Stake,
				providerEntry.Endpoints,
				providerEntry.Geolocation,
				providerEntry.Moniker,
				providerEntry.DelegateLimit,
				providerEntry.DelegateCommission,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().String(EndpointsFlagName, "", "The endpoints provider is offering in the format \"endpoint-url,geolocation endpoint-url,geolocation\"")
	cmd.Flags().String(AmountFlagName, "", "modify the provider's staked amount")
	cmd.Flags().Int32(GeolocationFlag, 0, "modify the provider's geolocation")
	cmd.Flags().Uint64(types.FlagCommission, 100, "The provider's commission from the delegators (default 100)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
