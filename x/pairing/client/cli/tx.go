package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"

	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/x/pairing/client/utils"
	"github.com/lavanet/lava/x/pairing/types"
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
		Args:  cobra.ExactArgs(1),
		Short: "Submit an unstake proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit an unstake proposal.
The proposal details must be supplied via a JSON file.

Example:
$ %s tx gov pairing-proposal unstake <path/to/proposal.json> --from=<key_or_address>
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			proposal, err := utils.ParseUnstakeProposalJSON(clientCtx.LegacyAmino, args[0])
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
