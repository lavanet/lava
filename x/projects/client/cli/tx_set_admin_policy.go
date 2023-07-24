package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdSetPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-policy [project-index] [policy-file-path]",
		Short: "set policy to a project",
		Long:  `The set-policy command allows a project admin to set a new policy to its project. The policy file is a YAML file (see cookbook/projects/example_policy.yml for reference). The new policy will be applied from the next epoch.`,
		Example: `required flags: --from <creator-address>
		lavad tx project set-policy [project-index] [policy-file-path] --from <creator_address>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			projectId := args[0]
			adminPolicyFilePath := args[1]
			policy, err := planstypes.ParsePolicyFromYaml(adminPolicyFilePath)
			if err != nil {
				return err
			}

			msg := types.NewMsgSetPolicy(
				clientCtx.GetFromAddress().String(),
				projectId,
				*policy,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}

func verifyChainPoliciesAreCorrectlySet(clientCtx client.Context, policy *planstypes.Policy) error {
	specQuerier := spectypes.NewQueryClient(clientCtx)
	var chainInfo *spectypes.QueryShowChainInfoResponse
	for policyIdx, chainPolicy := range policy.ChainPolicies {
		for idx, collection := range chainPolicy.Collections {
			if collection.AddOn == "" {
				// fix the addon for a collection on an optiona apiInterface
				if chainInfo == nil {
					var err error
					chainInfo, err = specQuerier.ShowChainInfo(context.Background(), &spectypes.QueryShowChainInfoRequest{ChainName: chainPolicy.ChainId})
					if err != nil {
						return err
					}
				}
				for _, optionalApiInterface := range chainInfo.OptionalInterfaces {
					if optionalApiInterface == collection.ApiInterface {
						policy.ChainPolicies[policyIdx].Collections[idx].AddOn = optionalApiInterface
						continue
					}
				}
				return fmt.Errorf("can't set an empty addon in a collection, empty addons are ignored %#v", chainPolicy)
			}
		}
	}
	return nil
}
