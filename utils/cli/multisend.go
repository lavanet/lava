package cli

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	commontypes "github.com/lavanet/lava/utils/common/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const HotWalletFlagName = "hot-wallet"

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

			clientCtxOrigin, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			authquerier := authtypes.NewQueryClient(clientCtxOrigin)

			hotWallet, err := cmd.Flags().GetString(HotWalletFlagName)
			clientCtxHotWallet := clientCtxOrigin
			useHotWallet := false
			if err != nil {
				return err
			}
			if hotWallet != "" {
				useHotWallet = true
				cmd.Flags().Set("from", hotWallet)
				cmd.Flags().Set("ledger", "false")
				clientCtxHotWallet, err = client.GetClientTxContext(cmd)
				if err != nil {
					return err
				}
			}

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

			getSequence := func(account string) (uint64, error) {
				res, err := authquerier.Account(cmd.Context(), &authtypes.QueryAccountRequest{Address: account})
				if err != nil {
					return 0, err
				}

				acc, ok := res.Account.GetCachedValue().(authtypes.AccountI)
				if !ok {
					return 0, fmt.Errorf("cant unmarshal")
				}

				return acc.GetSequence(), err
			}

			getResponse := func(account string, sequence uint64) bool {
				tmEvents := []string{
					fmt.Sprintf("%s.%s='%s/%d'", sdk.EventTypeTx, sdk.AttributeKeyAccountSequence, account, sequence),
				}
				txs, err := authtx.QueryTxsByEvents(clientCtxOrigin, tmEvents, query.DefaultPage, query.DefaultLimit, "")
				if err != nil {
					fmt.Printf("failed to query tx %s\n", err)
					return false
				}
				if len(txs.Txs) == 0 {
					fmt.Println("found no txs matching given address and sequence combination")
					return false
				}
				if len(txs.Txs) > 1 {
					// This case means there's a bug somewhere else in the code. Should not happen.
					fmt.Printf("found %d txs matching given address and sequence combination\n", len(txs.Txs))
					return false
				}

				return txs.Txs[0].Code == 0
			}

			waitSequenceChange := func(account string, sequence uint64) error {
				currentSequence, err := getSequence(account)
				if err != nil {
					return err
				}
				for currentSequence < sequence {
					fmt.Printf("waiting for sequence number %d current %d \n", sequence, currentSequence)
					time.Sleep(5 * time.Second)

					currentSequence, err = getSequence(account)
					if err != nil {
						return err
					}
				}
				if !getResponse(account, sequence-1) {
					return fmt.Errorf("transaction failed")
				}
				return nil
			}

			expectedSequenceOrigin, err := getSequence(clientCtxOrigin.FromAddress.String())
			if err != nil {
				return err
			}

			expectedSequenceHW := expectedSequenceOrigin
			if useHotWallet {
				expectedSequenceHW, err = getSequence(clientCtxHotWallet.FromAddress.String())
				if err != nil {
					return err
				}
			}

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
					if useHotWallet {
						msg := banktypes.NewMsgSend(clientCtxOrigin.FromAddress, clientCtxHotWallet.FromAddress, totalAmount.Add(sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(5))))
						err := tx.GenerateOrBroadcastTxCLI(clientCtxOrigin, cmd.Flags(), msg)
						if err != nil {
							fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
							fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
							return err
						}
						expectedSequenceOrigin++
						err = waitSequenceChange(clientCtxOrigin.FromAddress.String(), expectedSequenceOrigin)
						if err != nil {
							fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
							fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
							return err
						}
					}

					err = waitSequenceChange(clientCtxHotWallet.FromAddress.String(), expectedSequenceHW)
					if err != nil {
						fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
						fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
						return err
					}
					msg := banktypes.NewMsgMultiSend([]banktypes.Input{banktypes.NewInput(clientCtxHotWallet.FromAddress, totalAmount)}, output)
					err := tx.GenerateOrBroadcastTxCLI(clientCtxHotWallet, cmd.Flags(), msg)
					if err != nil {
						fmt.Printf("failed sending records from %d to %d\n", startIndex, i)
						fmt.Printf("please run again with: multi-send [file.csv] %d\n", startIndex)
						return err
					}
					expectedSequenceHW++
					startIndex = uint64(i) + 1
					output = []banktypes.Output{}
					totalAmount = sdk.Coins{}
				}
			}

			return nil
		},
	}

	cmd.Flags().String(HotWalletFlagName, "", "optional, hot wallet to be used as a middle point")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewMultiSendTxCmd returns a CLI command handler for creating a MsgMultiSend transaction.
// For a better UX this command is limited to send funds from one account to two or more accounts.
func NewQueryTotalGasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-gas address chainid",
		Short:   `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.`,
		Long:    `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.`,
		Example: "total-gas lava@... NEAR",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			authquerier := authtypes.NewQueryClient(clientCtx)

			account := args[0]
			chainid := args[1]

			getSequence := func(account string) (uint64, error) {
				res, err := authquerier.Account(cmd.Context(), &authtypes.QueryAccountRequest{Address: account})
				if err != nil {
					return 0, err
				}

				acc, ok := res.Account.GetCachedValue().(authtypes.AccountI)
				if !ok {
					return 0, fmt.Errorf("cant unmarshal")
				}

				return acc.GetSequence(), err
			}

			getResponse := func(account string, sequence uint64) *sdk.TxResponse {
				tmEvents := []string{
					fmt.Sprintf("%s.%s='%s/%d'", sdk.EventTypeTx, sdk.AttributeKeyAccountSequence, account, sequence),
				}
				txs, err := authtx.QueryTxsByEvents(clientCtx, tmEvents, query.DefaultPage, query.DefaultLimit, "")
				if err != nil {
					fmt.Printf("failed to query tx %s\n", err)
					return nil
				}
				if len(txs.Txs) == 0 {
					fmt.Println("found no txs matching given address and sequence combination")
					return nil
				}
				if len(txs.Txs) > 1 {
					// This case means there's a bug somewhere else in the code. Should not happen.
					fmt.Printf("found %d txs matching given address and sequence combination\n", len(txs.Txs))
					return nil
				}

				return txs.Txs[0]
			}

			now := time.Now().UTC()
			txtime := now
			totalgas := int64(0)
			sequence, _ := getSequence(account)
			layout := time.RFC3339
			for now.Sub(txtime) < 24*time.Hour {
				sequence--
				tx := getResponse(account, sequence)

				if strings.Contains(tx.RawLog, chainid) && strings.Contains(tx.RawLog, "MsgRelayPayment") {
					totalgas += tx.GasUsed
				}
				// Parse the time string
				txtime, err = time.Parse(layout, tx.Timestamp)
				if err != nil {
					return err
				}

				fmt.Printf("sequence %d, totalgas %d txdiff %f sec\n", sequence, totalgas, now.Sub(txtime).Seconds())
			}

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
