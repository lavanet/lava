package cli

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const DeletePolicyFlagName = "delete-policy"

func CmdSetPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-policy project-index [policy-file-path]",
		Short: "set policy to a project",
		Long:  `The set-policy command allows a project admin to set a new policy to its project. The policy file is a YAML file (see cookbook/projects/example_policy.yml for reference). The new policy will be applied from the next epoch. To define a geolocation in the policy file, use the available geolocations: ` + planstypes.PrintGeolocations(),
		Example: `required flags: --from <creator-address>
		lavad tx project set-policy [project-index] [policy-file-path] --from <creator_address>
		lavad tx project set-policy admin [policy-file-path] --from <creator_address> (use this for the default admin policy)
		lavad tx project set-policy [project-index] --delete-policy --from <creator_address>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			projectId := args[0]
			if projectId == "admin" {
				projectId = clientCtx.GetFromAddress().String() + "-admin"
			}

			// check if the command includes --delete-policy
			deletePolicyFlag := cmd.Flags().Lookup(DeletePolicyFlagName)
			if deletePolicyFlag == nil {
				return fmt.Errorf("%s flag wasn't found", DeletePolicyFlagName)
			}
			deletePolicy := deletePolicyFlag.Changed

			var policy *planstypes.Policy
			if !deletePolicy {
				if len(args) < 2 {
					return fmt.Errorf("not enough arguments")
				}
				adminPolicyFilePath := args[1]
				policy, err = planstypes.ParsePolicyFromYamlPath(adminPolicyFilePath)
				if err != nil {
					return err
				}

				err = verifyChainPoliciesAreCorrectlySet(clientCtx, policy)
				if err != nil {
					return err
				}
			}

			msg := types.NewMsgSetPolicy(
				clientCtx.GetFromAddress().String(),
				projectId,
				policy,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.Flags().Bool(DeletePolicyFlagName, false, "deletes the policy")

	return cmd
}

func verifyChainPoliciesAreCorrectlySet(clientCtx client.Context, policy *planstypes.Policy) error {
	specQuerier := spectypes.NewQueryClient(clientCtx)
	var chainInfo *spectypes.QueryShowChainInfoResponse
	for policyIdx, chainPolicy := range policy.ChainPolicies {
		for idx, requirement := range chainPolicy.Requirements {
			if requirement.Collection.AddOn == "" {
				// fix the addon for a collection on an optional apiInterface
				if chainInfo == nil {
					var err error
					chainInfo, err = specQuerier.ShowChainInfo(context.Background(), &spectypes.QueryShowChainInfoRequest{ChainName: chainPolicy.ChainId})
					if err != nil {
						return err
					}
				}
				for _, optionalApiInterface := range chainInfo.OptionalInterfaces {
					if optionalApiInterface == requirement.Collection.ApiInterface {
						policy.ChainPolicies[policyIdx].Requirements[idx].Collection.AddOn = optionalApiInterface
						continue
					}
				}
				if len(requirement.Extensions) == 0 {
					return fmt.Errorf("can't set an empty addon in a collection without extensions it means requirement is empty, empty requirements are ignored %#v", chainPolicy)
				}
			}
		}
	}
	return nil
}
