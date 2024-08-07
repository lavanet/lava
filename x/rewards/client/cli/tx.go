package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

var DefaultRelativePacketTimeoutTimestamp = uint64((time.Duration(10) * time.Minute).Nanoseconds())

const (
	flagPacketTimeoutTimestamp       = "packet-timeout-timestamp"
	listSeparator                    = ","
	expeditedFlagName                = "expedited"
	minIprpcCostFlagName             = "min-cost"
	addIprpcSubscriptionsFlagName    = "add-subscriptions"
	removeIprpcSubscriptionsFlagName = "remove-subscriptions"
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

	cmd.AddCommand(CmdFundIprpc())
	// this line is used by starport scaffolding # 1

	return cmd
}

// SetIprpcDataProposalHandler is the param change proposal handler.
var SetIprpcDataProposalHandler = govclient.NewProposalHandler(NewSubmitSetIprpcDataProposalTxCmd)

// NewSubmitSetIprpcDataProposalTxCmd returns a CLI command handler for creating
// a set-iprpc-data proposal governance transaction.
func NewSubmitSetIprpcDataProposalTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-iprpc-data <deposit> --min-cost 0ulava --add-subscriptions addr1,addr2 --remove-subscriptions addr3,addr4",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a set IPRPC data proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a set IPRPC data proposal along with an initial deposit.
The proposal details (IPRPC data to set) are supplied via optional flags. To set the minimum IPRPC cost, use the --min-cost flag.
To add IPRPC eligible subscriptions use the --add-subscriptions flag. To remove IPRPC eligible subscriptions use the --remove-subscriptions flag.
Optionally, you can make the proposal expedited using the --expedited flag.
Example:
$ %s tx gov submit-legacy-proposal set-iprpc-data --min-cost 0ulava --add-subscriptions addr1,addr2 --remove-subscriptions addr3,addr4
`,
				version.AppName,
			),
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

			// get min cost
			costStr, err := cmd.Flags().GetString(minIprpcCostFlagName)
			if err != nil {
				return err
			}
			cost, err := sdk.ParseCoinNormalized(costStr)
			if err != nil {
				return err
			}

			// get current iprpc subscriptions
			q := types.NewQueryClient(clientCtx)
			res, err := q.ShowIprpcData(context.Background(), &types.QueryShowIprpcDataRequest{})
			if err != nil {
				return err
			}
			subs := res.IprpcSubscriptions

			// add from msg
			subsToAdd, err := cmd.Flags().GetStringSlice(addIprpcSubscriptionsFlagName)
			if err != nil {
				return err
			}
			subs = append(subs, subsToAdd...)

			// remove duplicates
			iprpcSubs := []string{}
			unique := map[string]bool{}
			for _, sub := range subs {
				if !unique[sub] {
					unique[sub] = true
					iprpcSubs = append(iprpcSubs, sub)
				}
			}

			// remove from msg
			subsToRemove, err := cmd.Flags().GetStringSlice(removeIprpcSubscriptionsFlagName)
			if err != nil {
				return err
			}
			for _, sub := range subsToRemove {
				iprpcSubs, _ = lavaslices.Remove(iprpcSubs, sub)
			}

			deposit, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.MsgSetIprpcData{
				Authority:          authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				IprpcSubscriptions: iprpcSubs,
				MinIprpcCost:       cost,
			}

			submitPropMsg, err := govv1.NewMsgSubmitProposal([]sdk.Msg{&msg}, deposit, from.String(), "", "Set IPRPC data", "Set IPRPC data", isExpedited)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), submitPropMsg)
		},
	}
	cmd.Flags().String(minIprpcCostFlagName, "0ulava", "set minimum iprpc cost")
	cmd.Flags().StringSlice(addIprpcSubscriptionsFlagName, []string{}, "add iprpc eligible subscriptions")
	cmd.Flags().StringSlice(removeIprpcSubscriptionsFlagName, []string{}, "remove iprpc eligible subscriptions")
	cmd.Flags().Bool(expeditedFlagName, false, "set to true to make the spec proposal expedited")
	cmd.MarkFlagRequired(minIprpcCostFlagName)
	return cmd
}
