package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"

	// "github.com/cosmos/cosmos-sdk/client/flags"
	lavautils "github.com/lavanet/lava/v2/utils"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	dualstakingTypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/client/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

var DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator              = ","
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdStakeProvider())
	cmd.AddCommand(CmdBulkStakeProvider())
	cmd.AddCommand(CmdUnstakeProvider())
	cmd.AddCommand(CmdRelayPayment())
	cmd.AddCommand(CmdFreeze())
	cmd.AddCommand(CmdUnfreeze())
	cmd.AddCommand(CmdModifyProvider())
	cmd.AddCommand(CmdSimulateRelayPayment())

	// this line is used by starport scaffolding # 1

	return cmd
}

// NewSubmitUnstakeProposalTxCmd returns a CLI command handler for creating
// an unstake proposal governance transaction.
func NewSubmitUnstakeProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unstake proposal-file",
		Args:  cobra.RangeArgs(1, 4),
		Short: "Submit an unstake proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit an unstake proposal.
			The proposal details must be supplied via a JSON file.
			Example:
			$ %s tx gov pairing-proposal unstake <path/to/proposal.json> --from=<key_or_address>
			$ %s tx gov pairing-proposal unstake <provider-address> <chainid> <slash-factor> <deposit> --from=<key_or_address>
			`, version.AppName, version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			ctx := context.Background()
			if err != nil {
				return err
			}
			var content *types.UnstakeProposal
			var deposit sdk.Coins
			if len(args) == 1 {
				proposal, err := utils.ParseUnstakeProposalJSON(clientCtx.LegacyAmino, args[0])
				if err != nil {
					return err
				}
				content = &proposal.Proposal
				deposit, err = sdk.ParseCoinsNormalized(proposal.Deposit)
				if err != nil {
					return err
				}
			}
			if len(args) == 4 {
				deposit, err = sdk.ParseCoinsNormalized(args[3])
				if err != nil {
					return err
				}

				slashfactor, err := strconv.ParseUint(args[2], 10, 64)
				if err != nil || slashfactor == 0 || slashfactor > 100 {
					return lavautils.LavaFormatError("slashing factor needs to be an integer [1,100]", nil)
				}

				pairingQuerier := types.NewQueryClient(clientCtx)
				response, err := pairingQuerier.Providers(ctx, &types.QueryProvidersRequest{
					ChainID:    args[1],
					ShowFrozen: true,
				})
				if err != nil {
					return err
				}
				if len(response.StakeEntry) == 0 {
					return lavautils.LavaFormatError("provider isn't staked on chainID, no providers at all", nil)
				}
				var providerEntry *epochstoragetypes.StakeEntry
				for idx, provider := range response.StakeEntry {
					if provider.Address == args[0] {
						providerEntry = &response.StakeEntry[idx]
						break
					}
				}
				if providerEntry == nil {
					return lavautils.LavaFormatError("provider isn't staked on chainID, no address match", nil)
				}

				dualstakingQuerier := dualstakingTypes.NewQueryClient(clientCtx)
				delegators, err := dualstakingQuerier.ProviderDelegators(ctx, &dualstakingTypes.QueryProviderDelegatorsRequest{Provider: providerEntry.Address})
				if err != nil {
					return lavautils.LavaFormatError("failed to fetch delegators", nil)
				}

				content = &types.UnstakeProposal{}
				content.Title = "unstaking and slashing provider"
				content.Description = "unstaking and slashing provider and providers delegators by proposal"
				content.ProvidersInfo = []types.ProviderUnstakeInfo{{Provider: providerEntry.Address, ChainId: providerEntry.Chain}}
				content.DelegatorsSlashing = []types.DelegatorSlashing{}
				for _, delegator := range delegators.Delegations {
					if delegator.ChainID == providerEntry.Chain {
						content.DelegatorsSlashing = append(content.DelegatorsSlashing, types.DelegatorSlashing{
							Delegator:      delegator.Delegator,
							SlashingAmount: sdk.NewCoin(commontypes.TokenDenom, delegator.Amount.Amount.MulRaw(int64(slashfactor)).QuoRaw(100)),
						})
					}
				}
			}

			from := clientCtx.GetFromAddress()

			msg, err := v1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}
