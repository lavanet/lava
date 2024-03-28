package cli

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	commontypes "github.com/lavanet/lava/common/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cobra"
)

const (
	BULK_ARG_COUNT = 4
)

var _ = strconv.Itoa(0)

func CmdStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-provider [chain-id] [amount] [endpoint endpoint ...] [geolocation] [validator] --from <address> --provider-moniker <moniker>",
		Short: `stake a provider on the lava blockchain on a specific specification`,
		Long: `args:
		[chain-id] is the spec the provider wishes to support
		[amount] is the ulava amount to be staked, this is the final amount, if is lower than the the original stake it will unbond from the provider (and delegated validators)
		[endpoint endpoint ...] are a space separated list of HOST:PORT,geolocation,[apiInterface,apiInterface,addon,addon,] should be defined within "quotes". Note that you can specify the geolocation using a single uint64 (which will be treated as a bitmask (see plan.proto)) or using one of the geolocation codes. Valid geolocation codes: AF, AS, AU, EU, USE (US east), USC, USW, GL (global).
		[geolocation] should be the geolocation codes to be staked for. You can also use the geolocation codes syntax: EU,AU,AF,etc. Note that this geolocation should be a union of the endpoints' geolocations.
		[validator] delegate to a validator with the same amount with dualstaking, if not provided the validator will be chosen for best fit, when amount is decreasing to unbond, this determines the validator to extract funds from
		
		IMPORTANT: endpoint should not contain your node URL, it should point to the grpc listener of your provider service defined in your provider config or cli args`,
		Example: `
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com:2221,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider-africa.com:2221,AF my-provider-europe.com:2221,EU" AF,EU lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "LAV1" 500000ulava "my-provider.com:2221,1,tendermintrpc,rest,grpc" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "LAV1" 500000ulava "my-provider.com:2221,1,tendermintrpc,rest,grpc" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "LAV1" 500000ulava "my-provider.com:2221,1,tendermintrpc,rest,grpc,archive,trace" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,

		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainID := args[0]
			argAmount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			argEndpoints, argGeolocation, err := HandleEndpointsAndGeolocationArgs(strings.Fields(args[2]), args[3])
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			moniker, err := cmd.Flags().GetString(types.FlagMoniker)
			if err != nil {
				return err
			}

			commission, err := cmd.Flags().GetUint64(types.FlagCommission)
			if err != nil {
				return err
			}

			delegationLimitStr, err := cmd.Flags().GetString(types.FlagDelegationLimit)
			if err != nil {
				return err
			}
			delegationLimit, err := sdk.ParseCoinNormalized(delegationLimitStr)
			if err != nil {
				return err
			}

			validator := args[4]

			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				validator,
				argChainID,
				argAmount,
				argEndpoints,
				argGeolocation,
				moniker,
				delegationLimit,
				commission,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().Uint64(types.FlagCommission, 50, "The provider's commission from the delegators (default 50)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.MarkFlagRequired(types.FlagMoniker)
	cmd.MarkFlagRequired(types.FlagDelegationLimit)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdBulkStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk-stake-provider [chain-id,chain-id,chain-id...] [amount] [endpoint-url,geolocation endpoint-url,geolocation] [geolocation] {repeat arguments for another bulk} [validator] --from <address> --provider-moniker <moniker>",
		Short: "used to stake with a provider on a given endpoint in all of it's api interfaces and all chains with the same stake, each chain will require it's own stake, this command can be run in bulk, by repeating the arguments. for dual staking you can optionaly choose a validator to delegate",
		Long: `args:
		[chain-id,chain-id] is the specs the provider wishes to support separated by a ','
		[amount] is the ulava amount to be staked
		[endpoint-url,geolocation endpoint-url,geolocation...] are a space separated list of HOST:PORT,geolocation, should be defined within "quotes"
		[geolocation] should be the geolocation code to be staked for
		{repeat for another bulk} - creates a new Msg within the transaction with different arguments, can be used to run many changes in many chains without waiting for a new block
		[validator] validator address to delegate, if not provided the validator will be chosen for you for best match`,
		Example: `lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		bulk send: two bulks, listen for ETH1,LAV1 in one endpoint and OSMOSIS,COSMOSHUB in another
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 OSMOSIS,COSMOSHUB 500000ulava "my-other-grpc-addr.com:1111,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 OSMOSIS,COSMOSHUB 500000ulava "my-other-grpc-addr.com:1111,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,

		Args: func(cmd *cobra.Command, args []string) error {
			if len(args)%BULK_ARG_COUNT != 1 {
				return fmt.Errorf("invalid number of arguments, needs to be a multiple of %d, arg count: %d", BULK_ARG_COUNT, len(args))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			moniker, err := cmd.Flags().GetString(types.FlagMoniker)
			if err != nil {
				return err
			}

			commission, err := cmd.Flags().GetUint64(types.FlagCommission)
			if err != nil {
				return err
			}

			delegationLimitStr, err := cmd.Flags().GetString(types.FlagDelegationLimit)
			if err != nil {
				return err
			}
			delegationLimit, err := sdk.ParseCoinNormalized(delegationLimitStr)
			if err != nil {
				return err
			}

			handleBulk := func(cmd *cobra.Command, args []string, validator string) (msgs []sdk.Msg, err error) {
				if len(args) != BULK_ARG_COUNT {
					return nil, fmt.Errorf("invalid argument length %d should be %d", len(args), BULK_ARG_COUNT)
				}
				argChainIDs := args[0]
				chainIDs := strings.Split(argChainIDs, ",")
				argAmount, err := sdk.ParseCoinNormalized(args[1])
				if err != nil {
					return nil, err
				}

				allEndpoints, argGeolocation, err := HandleEndpointsAndGeolocationArgs(strings.Fields(args[2]), args[3])
				if err != nil {
					return nil, err
				}

				for _, chainID := range chainIDs {
					if chainID == "" {
						continue
					}

					msg := types.NewMsgStakeProvider(
						clientCtx.GetFromAddress().String(),
						validator,
						chainID,
						argAmount,
						allEndpoints,
						argGeolocation,
						moniker,
						delegationLimit,
						commission,
					)

					if msg.DelegateLimit.Denom != commontypes.TokenDenom {
						return nil, sdkerrors.Wrapf(types.DelegateLimitError, "Coin denomanator is not ulava")
					}

					if err := msg.ValidateBasic(); err != nil {
						return nil, err
					}
					msgs = append(msgs, msg)
				}
				return msgs, nil
			}
			msgs := []sdk.Msg{}
			validator := args[len(args)-1]
			for argRange := 0; argRange < len(args)-1; argRange += BULK_ARG_COUNT {
				if argRange+BULK_ARG_COUNT > len(args) {
					return fmt.Errorf("invalid argument count %d > %d", argRange+BULK_ARG_COUNT, len(args))
				}
				newMsgs, err := handleBulk(cmd, args[argRange:argRange+BULK_ARG_COUNT], validator)
				if err != nil {
					return err
				}
				msgs = append(msgs, newMsgs...)
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().Uint64(types.FlagCommission, 50, "The provider's commission from the delegators (default 50)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.MarkFlagRequired(types.FlagMoniker)
	cmd.MarkFlagRequired(types.FlagDelegationLimit)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func HandleEndpointsAndGeolocationArgs(endpArg []string, geoArg string) (endp []epochstoragetypes.Endpoint, geo int32, err error) {
	var endpointsGeoloc int32

	// handle endpoints
	for _, endpointStr := range endpArg {
		split := strings.Split(endpointStr, ",")
		if len(split) < 2 {
			return nil, 0, fmt.Errorf("invalid endpoint format: %s", endpointStr)
		}

		ipPort := split[0]
		_, _, err = net.SplitHostPort(ipPort)
		if err != nil {
			return nil, 0, err
		}

		geoloc, err := planstypes.ParseGeoEnum(split[1])
		if err != nil {
			return nil, 0, fmt.Errorf("invalid endpoint format: %w, format: %s", err, strings.Join(split, ";"))
		}

		if geoloc == int32(planstypes.Geolocation_GL) {
			// if global ("GL"), append the endpoint in all possible geolocations
			for _, geo := range planstypes.GetAllGeolocations() {
				geoInt := int32(geo)
				endpoint := epochstoragetypes.Endpoint{
					IPPORT:      ipPort,
					Geolocation: geoInt,
				}
				if len(split) > 2 {
					endpoint.Addons = split[2:]
				}
				endp = append(endp, endpoint)
			}
			endpointsGeoloc = int32(planstypes.Geolocation_GL)
		} else {
			// if not global, verify the geolocation is a single region and append as is
			if !planstypes.IsGeoEnumSingleBit(geoloc) {
				return nil, 0, fmt.Errorf("endpoint must include exactly one geolocation code: %s", split[1])
			}
			endpoint := epochstoragetypes.Endpoint{
				IPPORT:      ipPort,
				Geolocation: geoloc,
			}
			if len(split) > 2 {
				endpoint.Addons = split[2:]
			}
			endp = append(endp, endpoint)
			endpointsGeoloc |= geoloc
		}
	}
	if geoArg != "*" {
		// handle geolocation
		geo, err = planstypes.ParseGeoEnum(geoArg)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid geolocation format: %w", err)
		}

		if geo != endpointsGeoloc {
			return nil, 0, types.GeolocationNotMatchWithEndpointsError
		}
	}
	return endp, endpointsGeoloc, nil
}

func getValidator(clientCtx client.Context, provider string) string {
	q := stakingtypes.NewQueryClient(clientCtx)
	ctx := context.Background()
	resD, err := q.DelegatorValidators(ctx, &stakingtypes.QueryDelegatorValidatorsRequest{DelegatorAddr: provider})
	if err == nil && len(resD.Validators) > 0 {
		return resD.Validators[0].OperatorAddress
	}

	resV, err := q.Validators(ctx, &stakingtypes.QueryValidatorsRequest{})
	if err != nil {
		panic("failed to fetch list of validators")
	}
	validatorBiggest := resV.Validators[0]
	for _, validator := range resV.Validators {
		if validator.Tokens.GT(validatorBiggest.Tokens) {
			validatorBiggest = validator
		}
	}
	return validatorBiggest.OperatorAddress
}
