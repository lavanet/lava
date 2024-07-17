package cli

import (
	"fmt"
	"strings"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	utilslib "github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/spec/client/utils"
	"github.com/lavanet/lava/v2/x/spec/types"

	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
)

var DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

const (
	flagPacketTimeoutTimestamp = "packet-timeout-timestamp"
	listSeparator              = ","
	devTestFlagName            = "lava-dev-test"
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

// NewSubmitParamChangeProposalTxCmd returns a CLI command handler for creating
// a parameter change proposal governance transaction.
func NewSubmitSpecAddProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spec-add [proposal-file,proposal-file,...]",
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

			isExpedited, err := cmd.Flags().GetBool(expeditedFlagName)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()
			content := &proposal.Proposal
			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			devTest, err := cmd.Flags().GetBool(devTestFlagName)
			if err == nil && devTest {
				// modify the lava spec for dev tests
				for idx, spec := range content.Specs {
					if spec.Index == "LAV1" {
						utilslib.LavaFormatInfo("modified lava spec time for dev tests")
						content.Specs[idx].AverageBlockTime = (1 * time.Second).Milliseconds()
						for collection := range content.Specs[idx].ApiCollections {
							for verification := range content.Specs[idx].ApiCollections[collection].Verifications {
								if content.Specs[idx].ApiCollections[collection].Verifications[verification].Name == "chain-id" {
									content.Specs[idx].ApiCollections[collection].Verifications[verification].Values[0].ExpectedValue = "*"
								}
								if content.Specs[idx].ApiCollections[collection].Verifications[verification].Name == "pruning" {
									content.Specs[idx].ApiCollections[collection].Verifications = append(content.Specs[idx].ApiCollections[collection].Verifications[:verification], content.Specs[idx].ApiCollections[collection].Verifications[verification+1:]...)
								}
							}
						}
					}
				}
			}

			contentAny, err := codectypes.NewAnyWithValue(content)
			if err != nil {
				return err
			}

			msgExecLegacy := govv1.NewMsgExecLegacyContent(contentAny, authtypes.NewModuleAddress(govtypes.ModuleName).String())

			submitPropMsg, err := govv1.NewMsgSubmitProposal([]sdk.Msg{msgExecLegacy}, deposit, from.String(), proposal.Proposal.Description, proposal.Proposal.Title, "Add a new spec", isExpedited)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), submitPropMsg)
		},
	}
	cmd.Flags().Bool(devTestFlagName, false, "set to true to modify the average block time for lava spec")
	cmd.Flags().Bool(expeditedFlagName, false, "set to true to make the spec proposal expedited")
	return cmd
}
