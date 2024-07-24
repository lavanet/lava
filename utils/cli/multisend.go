package cli

import (
	"encoding/csv"
	"encoding/json"
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
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	HotWalletFlagName  = "hot-wallet"
	RPCRetriesFlagName = "rpc-retries"
)

const (
	PRG_ready = iota
	PRG_bank_send
	PRG_bank_send_verified
	PRG_multisend
)

type Progress struct {
	Index             int    `json:"Index"`
	Progress          int    `json:"Progress"`
	SequenceOrigin    uint64 `json:"SequenceOrigin"`
	SequenceHotWallet uint64 `json:"SequenceHotWallet"`
	HotWallet         string `json:"HotWallet"`
}

// NewMultiSendTxCmd returns a CLI command handler for creating a MsgMultiSend transaction.
// For a better UX this command is limited to send funds from one account to two or more accounts.
func NewMultiSendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multi-send [file.csv] <start index> --from [address]",
		Short: `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.`,
		Long: `Send funds from one account to two or more accounts as instructed in the csv file and divides to multiple messages.
				  expected csv is two columns, first column contains the address to send to, second colum is the amount to send in ulava`,
		Example: `lavad test multi-send output.csv --from alice
				  lavad test multi-send output.csv --from alice --ledger --hot-wallet bob`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			MAX_ADDRESSES := 3000

			clientCtxOrigin, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			authquerier := authtypes.NewQueryClient(clientCtxOrigin)

			homedir, _ := cmd.Flags().GetString("home")
			progressFile := homedir + "/multi_send_progress.csv"
			progress := loadProgress(progressFile)

			hotWallet, err := cmd.Flags().GetString(HotWalletFlagName)
			clientCtxHotWallet := clientCtxOrigin
			useHotWallet := false
			if err != nil {
				return err
			}
			if hotWallet != "" {
				if progress.HotWallet != "" && progress.HotWallet != hotWallet {
					return fmt.Errorf("using different hot wallet than saved in progress file")
				}
				useHotWallet = true
				cmd.Flags().Set("from", hotWallet)
				cmd.Flags().Set("ledger", "false")
				clientCtxHotWallet, err = client.GetClientTxContext(cmd)
				if err != nil {
					return err
				}
			}
			progress.HotWallet = hotWallet

			retries, _ := cmd.Flags().GetInt(RPCRetriesFlagName)

			// Open the CSV file
			file, err := os.Open(args[0])
			if err != nil {
				return err
			}
			defer file.Close()

			if len(args) == 2 {
				index, err := strconv.Atoi(args[1])
				if err != nil {
					return err
				}
				progress.Index = index - 1
			}

			feesStr, _ := cmd.Flags().GetString(flags.FlagFees)
			fees, err := sdk.ParseCoinsNormalized(feesStr)
			if err != nil {
				panic(err)
			}

			// Create a new CSV reader
			reader := csv.NewReader(file)

			// Read all records
			records, err := reader.ReadAll()
			if err != nil {
				return err
			}

			getSequence := func(account string) (uint64, error) {
				var err error
				var res *authtypes.QueryAccountResponse
				for i := 0; i < retries; i++ {
					res, err = authquerier.Account(cmd.Context(), &authtypes.QueryAccountRequest{Address: account})
					if err == nil {
						break
					}
				}
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

				var err error
				var txs *sdk.SearchTxsResult
				for i := 0; i < retries; i++ {
					txs, err = authtx.QueryTxsByEvents(clientCtxOrigin, tmEvents, query.DefaultPage, query.DefaultLimit, "")
					if err == nil {
						break
					}
				}
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

			if progress.SequenceOrigin == 0 {
				progress.SequenceOrigin, err = getSequence(clientCtxOrigin.FromAddress.String())
				if err != nil {
					return err
				}
			}

			if progress.SequenceHotWallet == 0 {
				progress.SequenceHotWallet = progress.SequenceOrigin
				if useHotWallet {
					progress.SequenceHotWallet, err = getSequence(clientCtxHotWallet.FromAddress.String())
					if err != nil {
						return err
					}
				}
			}

			output := []banktypes.Output{}
			totalAmount := sdk.Coins{}
			for i := progress.Index; i < len(records); i++ {
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
					fmt.Printf("failed sending records from %d to %d\n", progress.Index, i)
					fmt.Printf("please run again with: multi-send [file.csv] %d\n", progress.Index)
					return err
				}
				output = append(output, banktypes.NewOutput(toAddr, coins))
				totalAmount = totalAmount.Add(coins...)

				if (i+1)%MAX_ADDRESSES == 0 || (i+1) == len(records) {
					if useHotWallet {
						if progress.Progress == PRG_ready {
							fmt.Printf("*********************sending from origin to hotwallet %s*******************\n", totalAmount.Add(fees...).String())
							currentSequence, err := getSequence(clientCtxOrigin.FromAddress.String())
							if err != nil {
								return err
							}
							if currentSequence != progress.SequenceOrigin {
								return fmt.Errorf("unexpected sequence for account %s, current %d, expected %d", clientCtxOrigin.FromAddress.String(), currentSequence, progress.SequenceOrigin)
							}
							msg := banktypes.NewMsgSend(clientCtxOrigin.FromAddress, clientCtxHotWallet.FromAddress, totalAmount.Add(fees...))
							err = tx.GenerateOrBroadcastTxCLI(clientCtxOrigin, cmd.Flags(), msg)
							if err != nil {
								fmt.Printf("failed sending records from %d to %d\n", progress.Index, i)
								fmt.Printf("please run again with: multi-send [file.csv] %d\n", progress.Index)
								return err
							}
							progress.Progress = PRG_bank_send
							progress.SequenceOrigin++
							saveProgress(progress, progressFile)
						}
						if progress.Progress == PRG_bank_send {
							fmt.Printf("*********************verifing bank send *******************\n")
							err = waitSequenceChange(clientCtxOrigin.FromAddress.String(), progress.SequenceOrigin)
							if err != nil {
								fmt.Printf("failed sending records from %d to %d\n", progress.Index, i)
								fmt.Printf("please run again with: multi-send [file.csv] %d\n", progress.Index)
								return err
							}
							progress.Progress = PRG_bank_send_verified
							saveProgress(progress, progressFile)
							fmt.Printf("*********************verified bank send *******************\n")
						}
					}

					if progress.Progress == PRG_bank_send_verified || progress.Progress == PRG_ready {
						msg := banktypes.NewMsgMultiSend([]banktypes.Input{banktypes.NewInput(clientCtxHotWallet.FromAddress, totalAmount)}, output)
						fmt.Printf("*********************sending records from %d to %d, total tokens %s*******************\n", progress.Index, i, totalAmount.String())
						currentSequence, err := getSequence(clientCtxHotWallet.FromAddress.String())
						if err != nil {
							return err
						}
						if currentSequence != progress.SequenceHotWallet {
							return fmt.Errorf("unexpected sequence for account %s, current %d, expected %d", clientCtxHotWallet.FromAddress.String(), currentSequence, progress.SequenceHotWallet)
						}
						err = tx.GenerateOrBroadcastTxCLI(clientCtxHotWallet, cmd.Flags(), msg)
						if err != nil {
							fmt.Printf("failed sending records from %d to %d\n", progress.Index, i)
							fmt.Printf("please run again with: multi-send [file.csv] %d\n", progress.Index)
							return err
						}
						progress.Progress = PRG_multisend
						progress.SequenceHotWallet++
						saveProgress(progress, progressFile)
					}

					if progress.Progress == PRG_multisend {
						fmt.Printf("*********************verifing multi send *******************\n")
						err = waitSequenceChange(clientCtxHotWallet.FromAddress.String(), progress.SequenceHotWallet)
						if err != nil {
							fmt.Printf("failed sending records from %d to %d\n", progress.Index, i)
							fmt.Printf("please run again with: multi-send [file.csv] %d\n", progress.Index)
							return err
						}
						progress.Progress = PRG_ready
						progress.Index = i + 1
						saveProgress(progress, progressFile)
						fmt.Printf("*********************verified multi send *******************\n")
					}
					output = []banktypes.Output{}
					totalAmount = sdk.Coins{}
				}
			}

			return nil
		},
	}

	cmd.Flags().Int(RPCRetriesFlagName, 3, "number of retries on rpc error")
	cmd.Flags().String(HotWalletFlagName, "", "optional, hot wallet to be used as a middle point")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func saveProgress(progress Progress, filename string) {
	data, err := json.MarshalIndent(progress, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling progress:", err)
		return
	}

	err = os.WriteFile(filename, data, 0o644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}
}

func loadProgress(filename string) Progress {
	var progress Progress

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Println("File does not exist, using default progress")
		return Progress{Index: 0, Progress: PRG_ready} // Default values
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return progress
	}

	err = json.Unmarshal(data, &progress)
	if err != nil {
		fmt.Println("Error unmarshalling progress:", err)
	}

	return progress
}

func NewQueryTotalGasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "total-gas address chainid",
		Short:   `calculate the total gas used by a provider in 24H`,
		Long:    `calculate the total gas used by a provider in 24H`,
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
