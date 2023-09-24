package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

const (
	BULK_ARG_COUNT = 4
)

var _ = strconv.Itoa(0)

func CmdStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-provider [chain-id] [amount] [endpoint endpoint ...] [geolocation] --from <address> --provider-moniker <moniker>",
		Short: `stake a provider on the lava blockchain on a specific specification`,
		Long: `args:
		[chain-id] is the spec the provider wishes to support
		[amount] is the ulava amount to be staked
		[endpoint endpoint ...] are a space separated list of HOST:PORT,geolocation,[apiInterface,apiInterface,addon,addon,] should be defined within "quotes". Note that you can specify the geolocation using a single uint64 (which will be treated as a bitmask (see plan.proto)) or using one of the geolocation codes. Valid geolocation codes: AF, AS, AU, EU, USE (US east), USC, USW, GL (global).
		[geolocation] should be the geolocation codes to be staked for. You can also use the geolocation codes syntax: EU,AU,AF,etc. Note that this geolocation should be a union of the endpoints' geolocations.
		
		IMPORTANT: endpoint should not contain your node URL, it should point to the grpc listener of your provider service defined in your provider config or cli args`,
		Example: `
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com:2221,1" 1 -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider-africa.com:2221,AF" "my-provider-europe.com:2221,EU" AF,EU -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "LAV1" 500000ulava "my-provider.com:2221,1,tendermintrpc,rest,grpc" 1 -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "LAV1" 500000ulava "my-provider.com:2221,1,tendermintrpc,rest,grpc,archive,trace" 1 -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,

		Args: cobra.ExactArgs(4),
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

			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
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
	cmd.Flags().Uint64(types.FlagCommission, 100, "The provider's commission from the delegators (default 100)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.MarkFlagRequired(types.FlagMoniker)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdBulkStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk-stake-provider [chain-id,chain-id,chain-id...] [amount] [endpoint-url,geolocation endpoint-url,geolocation] [geolocation] {repeat arguments for another bulk} --from <address> --provider-moniker <moniker>",
		Short: "used to stake with a provider on a given endpoint in all of it's api interfaces and all chains with the same stake, each chain will require it's own stake, this command can be run in bulk, by repeating the arguments",
		Long: `args:
		[chain-id,chain-id] is the specs the provider wishes to support separated by a ','
		[amount] is the ulava amount to be staked
		[endpoint-url,geolocation endpoint-url,geolocation...] are a space separated list of HOST:PORT,geolocation, should be defined within "quotes"
		[geolocation] should be the geolocation code to be staked for
		{repeat for another bulk} - creates a new Msg within the transaction with different arguments, can be used to run many changes in many chains without waiting for a new block`,
		Example: `lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		bulk send: two bulks, listen for ETH1,LAV1 in one endpoint and COS3,COS5 in another
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 COS3,COS5 500000ulava "my-other-grpc-addr.com:1111,1" 1 -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args)%BULK_ARG_COUNT != 0 {
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

			handleBulk := func(cmd *cobra.Command, args []string) (msgs []sdk.Msg, err error) {
				if len(args) != BULK_ARG_COUNT {
					return nil, fmt.Errorf("invalid argument length %d should be %d", len(args), BULK_ARG_COUNT)
				}
				argChainIDs := args[0]
				chainIDs := strings.Split(argChainIDs, ",")
				argAmount, err := sdk.ParseCoinNormalized(args[1])
				if err != nil {
					return nil, err
				}
				tmpArg := strings.Fields(args[2])
				argEndpoints := map[int32]epochstoragetypes.Endpoint{}
				for _, endpointStr := range tmpArg {
					splitted := strings.Split(endpointStr, ",")
					if len(splitted) != 2 {
						return nil, fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,geolocation HOST:PORT,geolocation, received: %s", endpointStr)
					}
					geoloc, err := strconv.ParseInt(splitted[1], 10, 64)
					if err != nil {
						return nil, fmt.Errorf("invalid argument format in endpoints, geolocation must be a number")
					}
					geolocInt32 := int32(geoloc)
					endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], Geolocation: geolocInt32}
					argEndpoints[geolocInt32] = endpoint
				}
				argGeolocation, err := cast.ToInt32E(args[3])
				if err != nil {
					return nil, err
				}

				for _, chainID := range chainIDs {
					if chainID == "" {
						continue
					}
					allEndpoints := []epochstoragetypes.Endpoint{}
					for geoloc, endpointForGeoloc := range argEndpoints {
						endpoints := []epochstoragetypes.Endpoint{
							{IPPORT: endpointForGeoloc.IPPORT, Geolocation: geoloc},
						}
						allEndpoints = append(allEndpoints, endpoints...)
					}

					msg := types.NewMsgStakeProvider(
						clientCtx.GetFromAddress().String(),
						chainID,
						argAmount,
						allEndpoints,
						argGeolocation,
						moniker,
						delegationLimit,
						commission,
					)
					if err := msg.ValidateBasic(); err != nil {
						return nil, err
					}
					msgs = append(msgs, msg)
				}
				return msgs, nil
			}
			msgs := []sdk.Msg{}
			for argRange := 0; argRange < len(args); argRange += BULK_ARG_COUNT {
				if argRange+BULK_ARG_COUNT > len(args) {
					return fmt.Errorf("invalid argument count %d > %d", argRange+BULK_ARG_COUNT, len(args))
				}
				newMsgs, err := handleBulk(cmd, args[argRange:argRange+BULK_ARG_COUNT])
				if err != nil {
					return err
				}
				msgs = append(msgs, newMsgs...)
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().Uint64(types.FlagCommission, 100, "The provider's commission from the delegators (default 100)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.MarkFlagRequired(types.FlagMoniker)
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

		geoloc, err := planstypes.ParseGeoEnum(split[1])
		if err != nil {
			return nil, 0, fmt.Errorf("invalid endpoint format: %w, format: %s", err, strings.Join(split, ";"))
		}

		if geoloc == int32(planstypes.Geolocation_GL) {
			// if global ("GL"), append the endpoint in all possible geolocations
			for _, geo := range planstypes.GetAllGeolocations() {
				geoInt := int32(geo)
				endpoint := epochstoragetypes.Endpoint{
					IPPORT:      split[0],
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
				IPPORT:      split[0],
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
