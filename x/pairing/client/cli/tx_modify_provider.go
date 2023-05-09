package cli

import (
	"context"
	"fmt"
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
			address := clientKey.GetAddress().String()

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
				if provider.Address == address {
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
			newEndpointsStr, err := cmd.Flags().GetString(EndpointsFlagName)
			if err != nil {
				return err
			}
			if newEndpointsStr != "" {
				tmpArg := strings.Fields(newEndpointsStr)
				argEndpoints := []epochstoragetypes.Endpoint{}
				for _, endpointStr := range tmpArg {
					splitted := strings.Split(endpointStr, ",")
					if len(splitted) != 3 {
						return fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,useType,geolocation HOST:PORT,useType,geolocation, received: %s", endpointStr)
					}
					geoloc, err := strconv.ParseUint(splitted[2], 10, 64)
					if err != nil {
						return fmt.Errorf("invalid argument format in endpoints, geolocation must be a number")
					}
					endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1], Geolocation: geoloc}
					argEndpoints = append(argEndpoints, endpoint)
				}
				providerEntry.Endpoints = argEndpoints
			}
			geolocation, err := cmd.Flags().GetUint64(GeolocationFlag)
			if err != nil {
				return err
			}
			if geolocation != 0 {
				providerEntry.Geolocation = geolocation
			}
			moniker, err := cmd.Flags().GetString(types.FlagMoniker)
			if err != nil {
				return err
			}
			if moniker != "" {
				providerEntry.Moniker = moniker
			}
			// modify fields
			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				argChainID,
				providerEntry.Stake,
				providerEntry.Endpoints,
				providerEntry.Geolocation,
				providerEntry.Moniker,
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
	cmd.Flags().Uint64(GeolocationFlag, 0, "modify the provider's geolocation")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
