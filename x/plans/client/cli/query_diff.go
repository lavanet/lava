package cli

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/lavanet/lava/v5/x/plans/client/utils"
	"github.com/lavanet/lava/v5/x/plans/types"
	"github.com/spf13/cobra"
)

func CmdDiff() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff [plan-index-or-proposal-file] {plan-index-or-proposal-file}",
		Short: "Show what changed between two plans",
		Long: `Show a field-by-field diff between two plans. Each argument can be either a
plans-add proposal JSON file (e.g. cookbook/plans/gateway.json) or an index of a plan
on-chain. When given a single proposal file, each plan in it is compared against its
current on-chain version, showing exactly what the proposal would change.

Examples:
  lavad q plan diff cookbook/plans/gateway.json cookbook/plans/gateway-full.json
  lavad q plan diff explorer adventurer
  lavad q plan diff cookbook/plans/gateway-full.json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			queryPlan := func(index string) (types.Plan, error) {
				res, err := queryClient.Info(cmd.Context(), &types.QueryInfoRequest{PlanIndex: index})
				if err != nil {
					return types.Plan{}, fmt.Errorf("cannot fetch on-chain plan %s: %w", index, err)
				}
				return res.PlanInfo, nil
			}

			// an argument is either a path of a plans-add proposal file or an on-chain plan index
			resolvePlans := func(arg string) ([]types.Plan, error) {
				if _, err := os.Stat(arg); err == nil {
					proposal, err := utils.ParsePlansAddProposalJSON(arg)
					if err != nil {
						return nil, fmt.Errorf("cannot parse plans proposal file %s: %w", arg, err)
					}
					return proposal.Proposal.Plans, nil
				}
				plan, err := queryPlan(arg)
				if err != nil {
					return nil, err
				}
				return []types.Plan{plan}, nil
			}

			printDiff := func(prev, next types.Plan) error {
				changes := types.DiffPlans(prev, next)
				if len(changes) == 0 {
					return clientCtx.PrintString(fmt.Sprintf("plan %s: no changes\n", next.Index))
				}
				out := fmt.Sprintf("plan %s: %d change(s)\n", next.Index, len(changes))
				for _, change := range changes {
					out += fmt.Sprintf("  - %s\n", change)
				}
				return clientCtx.PrintString(out)
			}

			if len(args) == 1 {
				// single argument must be a proposal file: diff each of its plans
				// against the current on-chain version
				if _, err := os.Stat(args[0]); err != nil {
					return fmt.Errorf("a single argument must be a plans proposal file (to compare against the on-chain version): %s", args[0])
				}
				nextPlans, err := resolvePlans(args[0])
				if err != nil {
					return err
				}
				for _, next := range nextPlans {
					prev, err := queryPlan(next.Index)
					if err != nil {
						return err
					}
					if err := printDiff(prev, next); err != nil {
						return err
					}
				}
				return nil
			}

			prevPlans, err := resolvePlans(args[0])
			if err != nil {
				return err
			}
			nextPlans, err := resolvePlans(args[1])
			if err != nil {
				return err
			}

			// compare directly when there is one plan on each side (also covers
			// comparing two different plan indices); otherwise match by index
			if len(prevPlans) == 1 && len(nextPlans) == 1 {
				return printDiff(prevPlans[0], nextPlans[0])
			}

			prevByIndex := map[string]types.Plan{}
			for _, plan := range prevPlans {
				prevByIndex[plan.Index] = plan
			}
			for _, next := range nextPlans {
				prev, found := prevByIndex[next.Index]
				if !found {
					if err := clientCtx.PrintString(fmt.Sprintf("plan %s: only in %s\n", next.Index, args[1])); err != nil {
						return err
					}
					continue
				}
				if err := printDiff(prev, next); err != nil {
					return err
				}
				delete(prevByIndex, next.Index)
			}
			for _, prev := range prevPlans {
				if _, stillThere := prevByIndex[prev.Index]; stillThere {
					if err := clientCtx.PrintString(fmt.Sprintf("plan %s: only in %s\n", prev.Index, args[0])); err != nil {
						return err
					}
				}
			}
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
