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
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/spf13/cobra"
)

const (
	BULK_ARG_COUNT = 5
)

var _ = strconv.Itoa(0)

func CmdStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-provider [chain-id] [amount] [endpoint endpoint ...] [geolocation] [validator] --from <vault_address> --provider-moniker <moniker> --provider <provider_address> (optional)",
		Short: `stake a provider on the lava blockchain on a specific specification`,
		Long: `args:
		[chain-id] is the spec the provider wishes to support
		[amount] is the ulava amount to be staked, this is the final amount, if is lower than the the original stake it will unbond from the provider (and delegated validators)
		[endpoint endpoint ...] are a space separated list of HOST:PORT,geolocation,[apiInterface,apiInterface,addon,addon,] should be defined within "quotes". Note that you can specify the geolocation using a single uint64 (which will be treated as a bitmask (see plan.proto)) or using one of the geolocation codes. Valid geolocation codes: AF, AS, AU, EU, USE (US east), USC, USW, GL (global).
		[geolocation] should be the geolocation codes to be staked for. You can also use the geolocation codes syntax: EU,AU,AF,etc. Note that this geolocation should be a union of the endpoints' geolocations.
		[validator] delegate to a validator with the same amount with dualstaking, if not provided the validator will be chosen for best fit, when amount is decreasing to unbond, this determines the validator to extract funds from
		
		Note: The provider address is used when the user wishes to separate the account the holds its funds and the account that is used to operate the provider process (for enhance security). When specifying the provider address, it can be hardcoded or derived automatically from the keyring. If a provider address is not specified, the provider address will be the vault address (that stakes). If a provider address is specified and is different than the vault address, you may use the "--grant-provider-gas-fees-auth" flag to let the vault account pay for the provider gas fees.
		After that, to use the vault's funds for gas fees, use Cosmos' --fee-granter flag.
		IMPORTANT: endpoint should not contain your node URL, it should point to the grpc listener of your provider service defined in your provider config or cli args`,
		Example: `
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com:2221,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com:2221,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --provider lava@1mugd5um8csmrcuw8d69tvptrz6e5e5avxsrmz8 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing stake-provider "ETH1" 500000ulava "my-provider.com:2221,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from provider-wallet --provider-moniker "my-moniker" --provider alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

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

			providerFromFlag, err := cmd.Flags().GetString(types.FlagProvider)
			if err != nil {
				return err
			}

			provider, err := utils.ParseCLIAddress(clientCtx, providerFromFlag)
			if err != nil {
				return err
			}

			grantProviderGasFeesAuthFlagUsed, err := cmd.Flags().GetBool(types.FlagGrantFeeAuth)
			if err != nil {
				return err
			}
			var feeGrantMsg *feegrant.MsgGrantAllowance
			if grantProviderGasFeesAuthFlagUsed {
				feeGrantMsg, err = CreateGrantFeeMsg(clientCtx.GetFromAddress().String(), provider)
				if err != nil {
					return err
				}
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

			identity, err := cmd.Flags().GetString(types.FlagIdentity)
			if err != nil {
				return err
			}

			website, err := cmd.Flags().GetString(types.FlagWebsite)
			if err != nil {
				return err
			}

			securityContact, err := cmd.Flags().GetString(types.FlagSecurityContact)
			if err != nil {
				return err
			}

			descriptionDetails, err := cmd.Flags().GetString(types.FlagDescriptionDetails)
			if err != nil {
				return err
			}

			description := stakingtypes.NewDescription(moniker, identity, website, securityContact, descriptionDetails)

			msg := types.NewMsgStakeProvider(
				clientCtx.GetFromAddress().String(),
				validator,
				argChainID,
				argAmount,
				argEndpoints,
				argGeolocation,
				delegationLimit,
				commission,
				provider,
				description,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			msgs := []sdk.Msg{msg}
			if feeGrantMsg != nil {
				msgs = append(msgs, feeGrantMsg)
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}
	cmd.Flags().String(types.FlagMoniker, "", "The provider's moniker (non-unique name)")
	cmd.Flags().Uint64(types.FlagCommission, 50, "The provider's commission from the delegators (default 50)")
	cmd.Flags().String(types.FlagDelegationLimit, "0ulava", "The provider's total delegation limit from delegators (default 0)")
	cmd.Flags().String(types.FlagProvider, "", "The provider's operational address (address used to operate the provider process, default is vault address)")
	cmd.Flags().String(types.FlagIdentity, "", "The provider's identity")
	cmd.Flags().String(types.FlagWebsite, "", "The provider's website")
	cmd.Flags().String(types.FlagSecurityContact, "", "The provider's security contact info")
	cmd.Flags().String(types.FlagDescriptionDetails, "", "The provider's description details")
	cmd.Flags().Bool(types.FlagGrantFeeAuth, false, "Let the provider use the vault address' funds for gas fees")
	cmd.MarkFlagRequired(types.FlagMoniker)
	cmd.MarkFlagRequired(types.FlagDelegationLimit)
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdBulkStakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk-stake-provider [chain-id,chain-id,chain-id...] [amount] [endpoint-url,geolocation endpoint-url,geolocation] [geolocation] {repeat arguments for another bulk} [validator] --from <vault_address> --provider-moniker <moniker> --provider [provider1,provider2,...] (optional)",
		Short: "used to stake with a provider on a given endpoint in all of it's api interfaces and all chains with the same stake, each chain will require it's own stake, this command can be run in bulk, by repeating the arguments. for dual staking you can optionaly choose a validator to delegate",
		Long: `args:
		[chain-id,chain-id] is the specs the provider wishes to support separated by a ','
		[amount] is the ulava amount to be staked
		[endpoint-url,geolocation endpoint-url,geolocation...] are a space separated list of HOST:PORT,geolocation, should be defined within "quotes"
		[geolocation] should be the geolocation code to be staked for
		{repeat for another bulk} - creates a new Msg within the transaction with different arguments, can be used to run many changes in many chains without waiting for a new block
		[validator] validator address to delegate, if not provided the validator will be chosen for you for best match
		Note: The provider address can be hardcoded or derived automatically from the keyring. If a provider address is not specified, the provider address will be the vault address (that stakes). If a provider address is specified and is different than the vault address, you may use the "--grant-provider-gas-fees-auth" flag to let the vault account pay for the provider gas fees.
		After that, to use the vault's funds for gas fees, use Cosmos' --fee-granter flag.`,
		Example: `lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		bulk send: two bulks, listen for ETH1,LAV1 in one endpoint and OSMOSIS,COSMOSHUB in another
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 OSMOSIS,COSMOSHUB 500000ulava "my-other-grpc-addr.com:1111,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 OSMOSIS,COSMOSHUB 500000ulava "my-other-grpc-addr.com:1111,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from servicer1 --provider-moniker "my-moniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
		lavad tx pairing bulk-stake-provider ETH1,LAV1 500000ulava "my-provider-grpc-addr.com:9090,1" 1 OSMOSIS,COSMOSHUB 500000ulava "my-other-grpc-addr.com:1111,1" 1 lava@valoper13w8ffww0akdyhgls2umvvudce3jxzw2s7fwcnk -y --from servicer1 --provider-moniker "my-moniker" --provider lava@1jkg2l9k6h9yk0hsqvfamj6g7ehnkve3d6kvf7t,alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE`,

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

			providerFromFlag, err := cmd.Flags().GetString(types.FlagProvider)
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

			handleBulk := func(args []string, validator string) (msgs []sdk.Msg, err error) {
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

				identity, err := cmd.Flags().GetString(types.FlagIdentity)
				if err != nil {
					return nil, err
				}

				website, err := cmd.Flags().GetString(types.FlagWebsite)
				if err != nil {
					return nil, err
				}

				securityContact, err := cmd.Flags().GetString(types.FlagSecurityContact)
				if err != nil {
					return nil, err
				}

				descriptionDetails, err := cmd.Flags().GetString(types.FlagDescriptionDetails)
				if err != nil {
					return nil, err
				}

				description := stakingtypes.NewDescription(moniker, identity, website, securityContact, descriptionDetails)

				for _, chainID := range chainIDs {
					if chainID == "" {
						continue
					}
					provider := clientCtx.GetFromAddress().String()
					if providerFromFlag != "" {
						provider, err = utils.ParseCLIAddress(clientCtx, providerFromFlag)
						if err != nil {
							return nil, err
						}
					}

					grantProviderGasFeesAuthFlagUsed, err := cmd.Flags().GetBool(types.FlagGrantFeeAuth)
					if err != nil {
						return nil, err
					}
					var feeGrantMsg *feegrant.MsgGrantAllowance
					if grantProviderGasFeesAuthFlagUsed {
						feeGrantMsg, err = CreateGrantFeeMsg(clientCtx.GetFromAddress().String(), provider)
						if err != nil {
							return nil, err
						}
					}

					if feeGrantMsg != nil {
						msgs = append(msgs, feeGrantMsg)
					}

					msg := types.NewMsgStakeProvider(
						clientCtx.GetFromAddress().String(),
						validator,
						chainID,
						argAmount,
						allEndpoints,
						argGeolocation,
						delegationLimit,
						commission,
						provider,
						description,
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
				newMsgs, err := handleBulk(args[argRange:argRange+BULK_ARG_COUNT], validator)
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
	cmd.Flags().String(types.FlagProvider, "", "The provider's operational addresses (addresses that are used to operate the provider process. default is vault address)")
	cmd.Flags().String(types.FlagIdentity, "", "The provider's identity")
	cmd.Flags().String(types.FlagWebsite, "", "The provider's website")
	cmd.Flags().String(types.FlagSecurityContact, "", "The provider's security contact info")
	cmd.Flags().String(types.FlagDescriptionDetails, "", "The provider's description details")
	cmd.Flags().Bool(types.FlagGrantFeeAuth, false, "Let the provider use the vault address' funds for gas fees")
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

// CreateGrantFeeMsg cosntructs a feegrant GrantAllowance msg for specific TXs to allow the provider use the vault's funds for gas fees
func CreateGrantFeeMsg(granter string, grantee string) (*feegrant.MsgGrantAllowance, error) {
	if grantee == granter {
		// no need to grant allowance if the granter and grantee are the same (vault = provider)
		return nil, nil //nolint
	}
	granterAcc, err := sdk.AccAddressFromBech32(granter)
	if err != nil {
		return nil, utils.LavaFormatError("failed granting feegrant for gas fees for granter", err,
			utils.LogAttr("granter", granter),
		)
	}

	granteeAcc, err := sdk.AccAddressFromBech32(grantee)
	if err != nil {
		return nil, utils.LavaFormatError("failed granting feegrant for gas fees for grantee", err,
			utils.LogAttr("grantee", grantee),
		)
	}

	grant, err := feegrant.NewAllowedMsgAllowance(&feegrant.BasicAllowance{}, []string{
		// pairing module TXs
		sdk.MsgTypeURL(&types.MsgRelayPayment{}),
		sdk.MsgTypeURL(&types.MsgStakeProvider{}), // for modify-provider TX
		sdk.MsgTypeURL(&types.MsgFreezeProvider{}),
		sdk.MsgTypeURL(&types.MsgUnfreezeProvider{}),

		// conflict module TXs
		sdk.MsgTypeURL(&conflicttypes.MsgConflictVoteCommit{}),
		sdk.MsgTypeURL(&conflicttypes.MsgConflictVoteReveal{}),
	})
	if err != nil {
		return nil, utils.LavaFormatError("failed granting feegrant for gas fees", err,
			utils.LogAttr("granter", granter),
			utils.LogAttr("grantee", grantee),
		)
	}

	msg, err := feegrant.NewMsgGrantAllowance(grant, granterAcc, granteeAcc)
	if err != nil {
		return nil, utils.LavaFormatError("failed granting feegrant for gas fees", err,
			utils.LogAttr("granter", granter),
			utils.LogAttr("grantee", grantee),
			utils.LogAttr("grant", grant.String()),
		)
	}

	err = msg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	return msg, nil
}
