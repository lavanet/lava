package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	FlagMemoOnly = "memo-only"
)

func CmdQueryGenerateIbcIprpcTx() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-ibc-iprpc-tx [spec] [duration] [amount] [src-port] [src-channel] --from <alice> --node <node-URI>",
		Short: `Query for generating an ibc-transfer TX JSON for IPRPC funding over IBC`,
		Long: `Query for generating an ibc-transfer TX JSON for IPRPC funding over IBC. 
		The generated TX is a regular ibc-transfer TX with a custom memo. The memo holds the necessary information for a IPRPC over IBC request: 
		creator, spec ID the user wishes to fund, and duration of the fund in months.
		Note, the memo's spec ID field is not validated. A valid spec ID is one that is registered on-chain in the Lava blockchain. To 
		check whether a spec ID is valid, use the "lavad q spec show-chain-info" query.
		
		The query must be called with the following flags:
		 - "--from" flag which will be the ibc-transfer sender. Note, the TX should be signed by the sender.
		 - "--node" flag which should be the node from which the ibc-transfer TX originates (probably not a Lava node).
		
		To generate only the memo, so it can be embedded manually when sending an ibc-transfer, use the optional 
		--memo-only flag (in this case, the amount, src-port, and src-channel arguments will be ignored).`,
		Args: cobra.RangeArgs(2, 5),
		Example: `
		lavad q rewards generate-ibc-iprpc-tx ETH1 3 100ulava transfer channel-0 --from bob --node tcp://localhost:36657 > tx.json
		lavad tx sign tx.json --from bob > signedTx.json
		lavad tx broadcast signedTx.json --from bob --node tcp://localhost:36657
		`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			spec := args[0]
			duration := args[1]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			clientCtx.GenerateOnly = true
			sender := clientCtx.FromAddress.String()

			// create memo
			senderAddr, err := utils.ParseCLIAddress(clientCtx, sender)
			if err != nil {
				return err
			}
			memo, err := createIbcIprpcMemo(senderAddr, spec, duration)
			if err != nil {
				return err
			}

			// handle --memo-only
			memoOnly, err := cmd.Flags().GetBool(FlagMemoOnly)
			if err != nil {
				return err
			}
			if memoOnly {
				cmd.Println(escapeMemo(memo))
				return nil
			}

			// create ibc-transfer msg
			if len(args) != 5 {
				return fmt.Errorf("not enough arguments for generating TX")
			}

			clientQueryCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			amount, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}
			srcPort := args[3]
			srcChannel := args[4]
			height, err := getTimeoutHeight(clientQueryCtx, srcPort, srcChannel)
			if err != nil {
				return err
			}
			msg := ibctransfertypes.NewMsgTransfer(srcPort, srcChannel, amount, sender, types.IbcIprpcReceiverAddress().String(), height, 0, memo)

			// print constructed transfer message with custom memo
			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			fees, err := getFees(clientQueryCtx)
			if err != nil {
				return err
			}
			txf = txf.WithFees(fees)

			return txf.PrintUnsignedTx(clientCtx, msg)
		},
	}

	cmd.Flags().Bool(FlagMemoOnly, false, "Generate only an ibc-transfer packet's memo for IPRPC over IBC")
	flags.AddTxFlagsToCmd(cmd)
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.MarkFlagRequired(flags.FlagNode)
	return cmd
}

func createIbcIprpcMemo(creator string, spec string, durationStr string) (string, error) {
	duration, err := strconv.ParseUint(durationStr, 10, 64)
	if err != nil {
		return "", err
	}

	return types.CreateIprpcMemo(creator, spec, duration)
}

// getFees returns a default fee in the native token of the node from which the ibc-transfer is sent (example: 1osmo)
func getFees(clientQueryCtx client.Context) (string, error) {
	stakingQuerier := stakingtypes.NewQueryClient(clientQueryCtx)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := stakingQuerier.Params(timeoutCtx, &stakingtypes.QueryParamsRequest{})
	if err != nil {
		return "", err
	}

	fees := "1" + res.Params.BondDenom

	// sanity check
	_, err = sdk.ParseCoinNormalized(fees)
	if err != nil {
		return "", err
	}

	return fees, nil
}

// getTimeoutHeight a default timeout height for the node from which the ibc-transfer is sent from
func getTimeoutHeight(clientQueryCtx client.Context, portId string, channelId string) (clienttypes.Height, error) {
	ibcQuerier := ibcchanneltypes.NewQueryClient(clientQueryCtx)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientRes, err := ibcQuerier.ChannelClientState(timeoutCtx, &ibcchanneltypes.QueryChannelClientStateRequest{
		PortId:    portId,
		ChannelId: channelId,
	})
	if err != nil {
		return clienttypes.ZeroHeight(), err
	}

	var clientState exported.ClientState
	if err := clientQueryCtx.InterfaceRegistry.UnpackAny(clientRes.IdentifiedClientState.ClientState, &clientState); err != nil {
		return clienttypes.ZeroHeight(), err
	}

	clientHeight, ok := clientState.GetLatestHeight().(clienttypes.Height)
	if !ok {
		return clienttypes.ZeroHeight(), fmt.Errorf("invalid height type. expected type: %T, got: %T", clienttypes.Height{}, clientState.GetLatestHeight())

	}

	defaultTimeoutHeightStr := ibctransfertypes.DefaultRelativePacketTimeoutHeight
	defaultTimeoutHeight, err := clienttypes.ParseHeight(defaultTimeoutHeightStr)
	if err != nil {
		return clienttypes.ZeroHeight(), err
	}

	clientHeight.RevisionHeight += defaultTimeoutHeight.RevisionHeight
	clientHeight.RevisionNumber += defaultTimeoutHeight.RevisionNumber

	return clientHeight, nil
}

func escapeMemo(memo string) string {
	return fmt.Sprintf("%q", string(memo))
}
