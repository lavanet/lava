package cli

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
)

func NewMultiCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-check [file.csv]",
		Short: `queries the balances of an account against the csv file`,
		Long:  `queries the balances of an account against the csv file`,
		Example: `lavad test multi-check output.csv
				  lavad test multi-check output.csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			retries, _ := cmd.Flags().GetInt(RPCRetriesFlagName)
			bankQuerier := banktypes.NewQueryClient(clientCtx)

			// Open the CSV file
			file, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer file.Close()

			// Create a new CSV reader
			reader := csv.NewReader(file)

			// Read all records
			records, err := reader.ReadAll()
			if err != nil {
				return err
			}

			queryBalanceWithRetries := func(addr sdk.AccAddress) (*banktypes.QueryAllBalancesResponse, error) {
				for i := 0; i < retries; i++ {
					res, err := bankQuerier.AllBalances(cmd.Context(), &banktypes.QueryAllBalancesRequest{Address: addr.String()})
					if err == nil {
						// utils.LavaFormatDebug("query balance", utils.Attribute{Key: "address", Value: addr.String()}, utils.Attribute{Key: "balance", Value: res.Balances})
						return res, nil
					}
					// else {
					// 	utils.LavaFormatError("failed to query balance", err, utils.Attribute{Key: "address", Value: addr.String()})
					// }
				}
				return nil, err
			}

			recordsLen := len(records)
			lessThan := 0
			others := 0
			errors := 0
			for i := 0; i < len(records); i++ {
				fmt.Printf("\rProgress: %d/%d errors: %d, lessThan:%d, others:%d", i+1, recordsLen, errors, lessThan, others)
				coins, err := sdk.ParseCoinsNormalized(records[i][1])
				if err != nil {
					fmt.Printf("failed decoding coins record %d\n", i)
					return err
				}

				if coins.IsZero() {
					fmt.Printf("invalid coins record %d\n", i)
					return fmt.Errorf("must send positive amount")
				}
				toAddr, err := sdk.AccAddressFromBech32(records[i][0])
				if err != nil {
					return err
				}

				res, err := queryBalanceWithRetries(toAddr)
				if err != nil || res == nil {
					errors++
					// utils.LavaFormatError("failed to query balance", err, utils.Attribute{Key: "address", Value: toAddr.String()})
					continue
				}
				found, coin := coins.Find("ulava")
				if found {
					found, resCoin := res.Balances.Find("ulava")
					if found && !resCoin.IsNil() && resCoin.IsGTE(coin) {
						// this wallet has enough balance
						others++
					} else {
						// fmt.Printf("wallet %s has less than expected\n", toAddr.String())
						lessThan++
					}
				}
			}
			fmt.Printf("\n---- results ----\n\n")
			fmt.Printf("less than: %d others: %d\n", lessThan, others)
			return nil
		},
	}

	cmd.Flags().Int(RPCRetriesFlagName, 3, "number of retries on rpc error")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
