package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"

	// "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/plans/client/utils"
	"github.com/lavanet/lava/v2/x/plans/types"
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

	// this line is used by starport scaffolding # 1

	return cmd
}

// NewSubmitPlansAddProposalTxCmd returns a CLI command handler for creating
// a new plan proposal governance transaction.
func NewSubmitPlansAddProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plans-add [proposal-file,proposal-file,...]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a plans add proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a plans add proposal.
The proposal details must be supplied via a JSON file. For values that contains
objects, only non-empty fields will be updated.

IMPORTANT: Currently changes are evaluated but not validated, so it is
very important that any "value" change is valid (ie. correct type and within bounds)

Proper vetting of a plans add proposal should prevent this from happening
(no deposits should occur during the governance process), but it should be noted
regardless.

Example:
$ %s tx gov plans-proposal plans-add <path/to/proposal.json> --from=<key_or_address>
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := utils.ParsePlansAddProposalJSON(args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := &proposal.Proposal
			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			msg, err := v1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}

// NewSubmitPlansDelProposalTxCmd returns a CLI command handler for creating
// a delete plan proposal governance transaction.
func NewSubmitPlansDelProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plans-del [proposal-file,proposal-file,...]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a plans delete proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a plans delete proposal.
The proposal details must be supplied via a JSON file. For values that contains
objects, only non-empty fields will be updated.

IMPORTANT: Currently changes are evaluated but not validated, so it is
very important that any "value" change is valid (ie. correct type and within bounds)

Proper vetting of a plans add proposal should prevent this from happening
(no deposits should occur during the governance process), but it should be noted
regardless.

Example:
$ %s tx gov plans-proposal plans-del <path/to/proposal.json> --from=<key_or_address>
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := utils.ParsePlansDelProposalJSON(clientCtx.LegacyAmino, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := &proposal.Proposal
			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			msg, err := v1beta1.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}
