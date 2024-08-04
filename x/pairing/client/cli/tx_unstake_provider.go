package cli

import (
	"context"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/lavanet/lava/v2/utils"
	dualstakingclient "github.com/lavanet/lava/v2/x/dualstaking/client/cli"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdUnstakeProvider() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unstake-provider [chain-id,chain-id,chain-id...] [optional: validator]",
		Short: "unstake a provider staked on a specific specification on the lava blockchain initiating an un-stake period, funds are returned at the end of the period",
		Long: `args:
		[chain-id,chain-id] is the specs the provider wishes to stop supporting separated by a ','
		[validator] optional arg. this is the validator that will get its delegation decreased due to the unstake. if 
		no validator is specified, the validator from the largest delegation is picked`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argChainIDs := args[0]
			chainIDs := strings.Split(argChainIDs, ",")

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			var validator string
			if len(args) > 1 {
				validator = args[1]
			} else {
				validator = dualstakingclient.GetValidator(clientCtx)
			}

			msgs := []sdk.Msg{}
			for _, chainID := range chainIDs {
				if chainID == "" {
					continue
				}
				msg := types.NewMsgUnstakeProvider(
					clientCtx.GetFromAddress().String(),
					chainID,
					validator,
				)
				if err := msg.ValidateBasic(); err != nil {
					return err
				}
				msgs = append(msgs, msg)

				revokeGrantFeeMsg, err := CreateRevokeFeeGrantMsg(clientCtx, chainID)
				if err != nil {
					return err
				}
				if revokeGrantFeeMsg != nil {
					msgs = append(msgs, revokeGrantFeeMsg)
				}
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// CreateRevokeFeeGrantMsg constructs a feegrant RevokeAllowance msg to revoke the feegrant of the provider when the vault account unstakes
func CreateRevokeFeeGrantMsg(clientCtx client.Context, chainID string) (*feegrant.MsgRevokeAllowance, error) {
	ctx := context.Background()
	vault := clientCtx.GetFromAddress().String()

	// find stake entry to get provider
	pairingQuerier := types.NewQueryClient(clientCtx)
	response, err := pairingQuerier.Providers(ctx, &types.QueryProvidersRequest{
		ChainID:    chainID,
		ShowFrozen: true,
	})
	if err != nil {
		return nil, utils.LavaFormatError("failed revoking feegrant for gas fees. cannot get providers for chain", err,
			utils.LogAttr("chain_id", chainID),
		)
	}
	if len(response.StakeEntry) == 0 {
		return nil, utils.LavaFormatError("failed revoking feegrant for gas fees. provider isn't staked on chainID, no providers at all", nil,
			utils.LogAttr("chain_id", chainID),
		)
	}
	var providerEntry *epochstoragetypes.StakeEntry
	for idx, provider := range response.StakeEntry {
		if provider.Vault == vault {
			providerEntry = &response.StakeEntry[idx]
			break
		}
	}
	if providerEntry == nil {
		return nil, utils.LavaFormatError("failed revoking feegrant for gas fees. provider isn't staked on chainID, no address match", nil,
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("vault", vault),
		)
	}

	// construct revoke grant msg
	if vault == providerEntry.Address {
		// when vault = provider there is no grant, do nothing
		return nil, nil //nolint
	}
	granterAcc, err := sdk.AccAddressFromBech32(vault)
	if err != nil {
		return nil, utils.LavaFormatError("failed revoking feegrant for gas fees for granter", err,
			utils.LogAttr("granter", vault),
		)
	}

	granteeAcc, err := sdk.AccAddressFromBech32(providerEntry.Address)
	if err != nil {
		return nil, utils.LavaFormatError("failed revoking feegrant for gas fees for grantee", err,
			utils.LogAttr("grantee", providerEntry.Address),
		)
	}

	feegrantQuerier := feegrant.NewQueryClient(clientCtx)
	res, err := feegrantQuerier.Allowance(ctx, &feegrant.QueryAllowanceRequest{Granter: vault, Grantee: providerEntry.Address})
	if err != nil {
		return nil, utils.LavaFormatError("failed querying feegrant for gas fees for granter", err,
			utils.LogAttr("granter", vault),
		)
	}

	if res.Allowance == nil {
		// no allowance found, do nothing
		return nil, nil //nolint
	}

	msg := feegrant.NewMsgRevokeAllowance(granterAcc, granteeAcc)

	return &msg, nil
}
