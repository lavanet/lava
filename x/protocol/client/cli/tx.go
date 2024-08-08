package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/lavanet/lava/v2/x/protocol/types"
)

var DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	expeditedFlagName          = "expedited"
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

// SetProtocolVersionProposalHandler is the param change proposal handler.
var SetProtocolVersionProposalHandler = govclient.NewProposalHandler(NewSubmitSetProtocolVersionProposalTxCmd)

// NewSubmitSetProtocolVersionProposalTxCmd returns a CLI command handler for creating
// a set-version proposal governance transaction.
func NewSubmitSetProtocolVersionProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-protocol-version target minimum <deposit>",
		Args:  cobra.ExactArgs(3),
		Short: "Submit a set version proposal",
		Long: strings.TrimSpace(
			`Submit a set protocol version proposal along with an initial deposit. The proposal sets the version of the lavap binary.
			Example:
			tx gov submit-legacy-proposal set-protocol-version v0.35.0 v0.32.1 10000000ulava`,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			from := clientCtx.GetFromAddress()

			isExpedited, err := cmd.Flags().GetBool(expeditedFlagName)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(args[2])
			if err != nil {
				return err
			}

			msg := types.MsgSetVersion{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Version:   &types.Version{ProviderTarget: args[0], ProviderMin: args[1], ConsumerTarget: args[0], ConsumerMin: args[1]},
			}

			submitPropMsg, err := govv1.NewMsgSubmitProposal([]sdk.Msg{&msg}, deposit, from.String(), "", "Set protocol version", "Set protocol version", isExpedited)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), submitPropMsg)
		},
	}

	cmd.Flags().Bool(expeditedFlagName, false, "set to true to make the spec proposal expedited")
	return cmd
}
