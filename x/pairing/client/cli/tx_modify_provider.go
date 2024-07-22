package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/spf13/cobra"
)

const (
	AmountFlagName    = "amount"
	EndpointsFlagName = "endpoints"
	GeolocationFlag   = "geolocation"
	ValidatorFlag     = "validator"
)

var _ = strconv.Itoa(0)

type flexibleFlag struct {
	Value interface{}
}

func (f *flexibleFlag) Type() string {
	return "flexible"
}

func (f *flexibleFlag) String() string {
	if s, ok := f.Value.(string); ok {
		return s
	} else if i, ok := f.Value.(int32); ok {
		return strconv.Itoa(int(i))
	}
	return ""
}

func (f *flexibleFlag) Set(value string) error {
	// Try to parse the value as an int32, and if it fails, store it as a string
	i, err := strconv.ParseInt(value, 10, 32)
	if err == nil {
		f.Value = int32(i)
	} else {
		f.Value = value
	}
	return nil
}

func CmdModifyProvider() *cobra.Command {
	var geolocationVar flexibleFlag

	cmd := &cobra.Command{
		Use:   "modify-provider [chain-id] --from <address>",
		Short: `modify a staked provider on the lava blockchain on a specific specification, provider must be already staked`,
		Long: `args:
		[chain-id] is the spec the provider wishes to modify the entry for
		`,
		Example: `lavad tx pairing modify-provider "ETH1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --from <wallet>
		lavad tx pairing modify-provider "ETH1" --endpoints "my-provider-africa.com:443,AF my-provider-europe.com:443,EU" --geolocation "AF,EU" --validator lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk --from <wallet>`,
		Args: cobra.ExactArgs(1),
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
				if provider.IsAddressVaultOrProvider(address.String()) {
					providerEntry = &response.StakeEntry[idx]
					break
				}
			}
			if providerEntry == nil {
				return utils.LavaFormatError("provider isn't staked on chainID, no address match", nil)
			}

			var validator string
			newAmount, err := cmd.Flags().GetString(AmountFlagName)
			if err != nil {
				return err
			}
			if newAmount != "" {
				newStake, err := sdk.ParseCoinNormalized(newAmount)
				if err != nil {
					return err
				}

				if !providerEntry.Stake.IsEqual(newStake) {
					if cmd.Flags().Changed(ValidatorFlag) {
						validator, err = cmd.Flags().GetString(types.FlagMoniker)
						if err != nil {
							return err
						}
					} else {
						return fmt.Errorf("increasing or decreasing stake must be accompanied with validator flag")
					}
				} else {
					validator = getValidator(clientCtx, clientCtx.GetFromAddress().String())
				}
				providerEntry.Stake = newStake
			}
			var geolocation int32
			if cmd.Flags().Changed(GeolocationFlag) {
				geolocation, err = planstypes.ParseGeoEnum(geolocationVar.String())
				if err != nil {
					return err
				}
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
				providerEntry.Description.Moniker = moniker
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

			identity, err := cmd.Flags().GetString(types.FlagIdentity)
			if err != nil {
				return err
			}
			if identity != "" {
				providerEntry.Description.Identity = identity
			}

			website, err := cmd.Flags().GetString(types.FlagWebsite)
			if err != nil {
				return err
			}
			if website != "" {
				providerEntry.Description.Website = website
			}

			securityContact, err := cmd.Flags().GetString(types.FlagSecurityContact)
			if err != nil {
				return err
			}
			if securityContact != "" {
				providerEntry.Description.SecurityContact = securityContact
			}

			descriptionDetails, err := cmd.Flags().GetString(types.FlagDescriptionDetails)
			if err != nil {
				return err
			}
			if descriptionDetails != "" {
				providerEntry.Description.Details = descriptionDetails
			}

			description, err := providerEntry.Description.EnsureLength()
			if err != nil {
				return err
			}

			// modify fields
			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				validator,
				argChainID,
				providerEntry.Stake,
				providerEntry.Endpoints,
				providerEntry.Geolocation,
				providerEntry.DelegateLimit,
				providerEntry.DelegateCommission,
				providerEntry.Address,
				description,
			)

			if msg.DelegateLimit.Denom != commontypes.TokenDenom {
				return sdkerrors.Wrapf(types.DelegateLimitError, "Coin denomanator is not ulava")
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().String(EndpointsFlagName, "", "The endpoints provider is offering in the format \"endpoint-url,geolocation endpoint-url,geolocation\"")
	cmd.Flags().String(AmountFlagName, "", "modify the provider's staked amount")
	cmd.Flags().String(ValidatorFlag, "", "the validator to delegate/bond to with dualstaking")
	cmd.Flags().Var(&geolocationVar, GeolocationFlag, `modify the provider's geolocation int32 or string value "EU,US"`)
	cmd.Flags().Uint64(types.FlagCommission, 50, "The provider's commission from the delegators (default 50)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.Flags().String(types.FlagIdentity, "", "The provider's identity")
	cmd.Flags().String(types.FlagWebsite, "", "The provider's website")
	cmd.Flags().String(types.FlagSecurityContact, "", "The provider's security contact info")
	cmd.Flags().String(types.FlagDescriptionDetails, "", "The provider's description details")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
