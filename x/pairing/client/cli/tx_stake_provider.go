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
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
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
		[endpoint endpoint ...] are a space separated list of HOST:PORT,useType,geolocation, should be defined within "quotes". Note that you can specify the geolocation using a single uint64 (which will be treated as a bitmask (see plan.proto)) or using one of the geolocation codes. Valid geolocation codes: AF, AS, AU, EU, USE (US east), USC, USW, GL (global).
		[geolocation] should be the geolocation codes to be staked for. You can also use the geolocation codes syntax: EU,AU,AF. Note that this geolocation should be a union of the endpoints' geolocations.`,
		Example: `
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com/rpc,jsonrpc,1" 1 -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com/rpc,jsonrpc,AF" "my-provider.com/rpc,jsonrpc,EU" AF,EU -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,
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

			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				argChainID,
				argAmount,
				argEndpoints,
				argGeolocation,
				moniker,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
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
			specQuerier := spectypes.NewQueryClient(clientCtx)
			allChains, err := specQuerier.ShowAllChains(context.Background(), &spectypes.QueryShowAllChainsRequest{})
			if err != nil {
				return utils.LavaFormatError("could not get the list of all chains, in order to construct the transaction", err)
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
				argEndpoints := map[uint64]epochstoragetypes.Endpoint{}
				for _, endpointStr := range tmpArg {
					splitted := strings.Split(endpointStr, ",")
					if len(splitted) != 2 {
						return nil, fmt.Errorf("invalid argument format in endpoints, must be: HOST:PORT,geolocation HOST:PORT,geolocation, received: %s", endpointStr)
					}
					geoloc, err := strconv.ParseUint(splitted[1], 10, 64)
					if err != nil {
						return nil, fmt.Errorf("invalid argument format in endpoints, geolocation must be a number")
					}
					endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: "STUB", Geolocation: geoloc}
					argEndpoints[geoloc] = endpoint
				}
				argGeolocation, err := cast.ToUint64E(args[3])
				if err != nil {
					return nil, err
				}

				chainsToEndpointsMap := map[string][]string{}
				for _, chainStructInfo := range allChains.ChainInfoList {
					chainsToEndpointsMap[chainStructInfo.ChainID] = chainStructInfo.EnabledApiInterfaces
				}
				for _, chainID := range chainIDs {
					if chainID == "" {
						continue
					}
					interfacesForThisChainID := chainsToEndpointsMap[chainID]
					allEndpoints := []epochstoragetypes.Endpoint{}
					for geoloc, endpointForGeoloc := range argEndpoints {
						endpoints := make([]epochstoragetypes.Endpoint, len(interfacesForThisChainID))
						for idx, interfaceName := range interfacesForThisChainID {
							endpoints[idx] = epochstoragetypes.Endpoint{IPPORT: endpointForGeoloc.IPPORT, Geolocation: geoloc, UseType: interfaceName}
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
	cmd.MarkFlagRequired(types.FlagMoniker)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func HandleEndpointsAndGeolocationArgs(endpArg []string, geoArg string) (endp []epochstoragetypes.Endpoint, geo uint64, err error) {
	for _, endpointStr := range endpArg {
		splitted := strings.Split(endpointStr, ",")
		if len(splitted) != 3 {
			return nil, 0, fmt.Errorf("invalid argument format in endpoints, received: %s", endpointStr)
		}

		// geolocation of an endpoint can be uint or a string representing a single geo region
		var geoloc uint64
		geoloc, valid := types.IsValidGeoEnum(splitted[2])
		if !valid {
			geoloc, err = strconv.ParseUint(splitted[2], 10, 64)
			if err != nil {
				return nil, 0, fmt.Errorf("invalid argument format in endpoints, geolocation must be a number or valid geolocation string")
			}
		}

		// if the user specified global ("GL"), append the endpoint in all possible geolocations
		if geoloc == uint64(planstypes.Geolocation_GL) {
			for geoName, geoVal := range planstypes.Geolocation_value {
				if geoName == planstypes.Geolocation_GL.String() || geoName == planstypes.Geolocation_GLS.String() {
					continue
				}
				endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1], Geolocation: uint64(geoVal)}
				endp = append(endp, endpoint)
			}
		} else {
			// if it's not global, verify that the endpoint's geolocation represents a single geo region
			geoRegions, _ := types.ExtractGeolocations(geoloc)
			if len(geoRegions) != 1 {
				return nil, 0, fmt.Errorf("invalid geolocation for endpoint, must represent one region")
			}
			endpoint := epochstoragetypes.Endpoint{IPPORT: splitted[0], UseType: splitted[1], Geolocation: geoloc}
			endp = append(endp, endpoint)
		}
	}

	// handle the case of string geolocation (example: "EU,AF,AS")
	splitted := strings.Split(geoArg, ",")
	strEnums := commontypes.RemoveDuplicatesFromSlice(splitted)
	for _, s := range strEnums {
		g, valid := types.IsValidGeoEnum(s)
		if valid {
			// if one of the endpoints is global, assign global value and break
			if g == uint64(planstypes.Geolocation_GL) {
				geo = types.GetCurrentGlobalGeolocation()
				break
			}
			// for non-global geolocations constructs the final geolocation uint value (addition works because we
			// removed duplicates and the geo regions represent a bitmap)
			geo += g
		}
	}

	// geolocation is not a list of enums, try to parse it as an uint
	if geo == 0 {
		geo, err = cast.ToUint64E(geoArg)
		if err != nil {
			return nil, 0, err
		}
	}
	return endp, geo, nil
}
