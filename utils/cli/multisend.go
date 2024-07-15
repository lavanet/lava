package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

// NewMultiSendTxCmd returns a CLI command handler for creating a MsgMultiSend transaction.
// For a better UX this command is limited to send funds from one account to two or more accounts.
func NewMultiSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "multi-send [file.csv] <start index> --from [address]",
		Short:   `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.`,
		Long:    `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.`,
		Example: "multi-send [file.csv] --from you",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			MAX_ADDRESSES := 3000

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			authquerier := authtypes.NewQueryClient(clientCtx)

			// Open the CSV file
			file, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer file.Close()

			startIndex := uint64(0)
			if len(args) == 2 {
				startIndex, err = strconv.ParseUint(args[1], 10, 64)
				if err != nil {
					return err
				}
				startIndex--
			}

			// Create a new CSV reader
			reader := csv.NewReader(file)

			// Read all records
			records, err := reader.ReadAll()
			if err != nil {
				return err
			}

			GetSequence := func() (uint64, error) {
				res, err := authquerier.Account(cmd.Context(), &authtypes.QueryAccountRequest{Address: clientCtx.FromAddress.String()})
				if err != nil {
					return 0, err
				}

				acc, ok := res.Account.GetCachedValue().(authtypes.AccountI)
				if !ok {
					return 0, fmt.Errorf("cant unmarshal")
				}

				return acc.GetSequence(), err
			}

			expectedSequence, err := GetSequence()
			if err != nil {
				return err
			}
			currentSequence := expectedSequence
			output := []banktypes.Output{}
			totalAmount := sdk.Coins{}
			records = records[startIndex:]
			for i, record := range records {
				coins, err := sdk.ParseCoinsNormalized(record[1])
				if err != nil {
					fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
					fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
					return err
				}

				if coins.IsZero() {
					fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
					fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
					return fmt.Errorf("must send positive amount")
				}
				toAddr, err := sdk.AccAddressFromBech32(record[0])
				if err != nil {
					fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
					fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
					return err
				}
				output = append(output, banktypes.NewOutput(toAddr, coins))
				totalAmount = totalAmount.Add(coins...)

				if (i+1)%MAX_ADDRESSES == 0 || (i+1) == len(records) {
					for currentSequence < expectedSequence {
						fmt.Printf("waiting for sequence number %d current %d \n", expectedSequence, currentSequence)
						time.Sleep(5 * time.Second)

						currentSequence, err = GetSequence()
						if err != nil {
							fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
							fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
							return err
						}
					}

					msg := banktypes.NewMsgMultiSend([]banktypes.Input{banktypes.NewInput(clientCtx.FromAddress, totalAmount)}, output)
					err := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
					if err != nil {
						fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
						fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
						return err
					}
					expectedSequence++
					startIndex = uint64(i) + 1
					output = []banktypes.Output{}
					totalAmount = sdk.Coins{}
				}
			}

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
