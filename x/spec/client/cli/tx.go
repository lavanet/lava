package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/spec/client/utils"
	"github.com/lavanet/lava/x/spec/types"

	"strings"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

var (
	DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())
)

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

// NewSubmitParamChangeProposalTxCmd returns a CLI command handler for creating
// a parameter change proposal governance transaction.
func NewSubmitSpecAddProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec-add [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a spec add proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a parameter proposal along with an initial deposit.
The proposal details must be supplied via a JSON file. For values that contains
objects, only non-empty fields will be updated.

IMPORTANT: Currently  changes are evaluated but not validated, so it is
very important that any "value" change is valid (ie. correct type and within bounds)

Proper vetting of a spec add proposal should prevent this from happening
(no deposits should occur during the governance process), but it should be noted
regardless.

Example:
$ %s tx gov spec-proposal spec-add <path/to/proposal.json> --from=<key_or_address>
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := utils.ParseSpecAddProposalJSON(clientCtx.LegacyAmino, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := types.NewSpecAddProposal(proposal.Title, proposal.Description, proposal.ToSpecs())
			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}

// NewSubmitParamChangeProposalTxCmd returns a CLI command handler for creating
// a parameter change proposal governance transaction.
func NewSubmitSpecModifyProposalTxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "spec-modify [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a spec modify proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a parameter proposal along with an initial deposit.
The proposal details must be supplied via a JSON file. For values that contains
objects, only non-empty fields will be updated.

IMPORTANT: Currently  changes are evaluated but not validated, so it is
very important that any "value" change is valid (ie. correct type and within bounds)

Proper vetting of a spec modify proposal should prevent this from happening
(no deposits should occur during the governance process), but it should be noted
regardless.

Example:
$ %s tx gov spec-proposal spec-modify <path/to/proposal.json> --from=<key_or_address>
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := utils.ParseSpecAddProposalJSON(clientCtx.LegacyAmino, args[0])
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := types.NewSpecModifyProposal(proposal.Title, proposal.Description, proposal.ToSpecs())
			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
}
