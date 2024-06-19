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
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/lavanet/lava/x/rewards/types"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

const (
	FlagTitle     = "title"
	FlagSummary   = "summary"
	FlagExpedited = "expedited"
)

func CmdTxSubmitIbcIprpcProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-ibc-iprpc-proposal [spec] [duration] [amount] [src-port] [src-channel] --from <alice> --node <node-URI> [flags]",
		Short: "TX for submitting an ibc-transfer proposal for IPRPC funding over IBC",
		Long: `TX for submitting an ibc-transfer proposal for IPRPC funding over IBC. 
		The generated proposal holds several transactions that when executed, an IPRPC over IBC transaction is triggered.
		The executed transactions (by order):
		1. Grant IBC transfer authz from the creator (alice) to the gov module.
		2. Grant Revoke authz from the creator (alice) to the gov module.
		3. Submit a proposal that holds the following transactions (by order):
		   3.1. Community pool spend transaction to the creator.
		   3.2. Authz Exec transaction with the following transactions (by order):
		      a. IBC transfer transaction with a custom memo (to trigger IPRPC over IBC).
			  b. Authz revoke transaction to revoke the grant from step 1.
			  b. Authz revoke transaction to revoke the grant from step 2.
		
		Arguments:
		 - spec: the spec ID which the user wants to fund. Note, this argument is not validated. A valid spec ID is one 
		that is registered on-chain in the Lava blockchain. To check whether a spec ID is valid, use the 
		"lavad q spec show-chain-info" query.
		 - duration: the duration in months that the user wants the IPRPC to last.
		 - amount: the total amount of coins to fund the IPRPC pool. Note that each month, the IPRPC pool will be 
		funded with amount/duration coins.
		 - src-port: source IBC port (usually "transfer").
		 - src-channel: source IBC channel.

		Important notes:
		  1. The "--node" flag is mandatory. It should hold the node URI from which the ibc-transfer transaction 
		  originates (probably not a Lava node).
		  2. After the proposal passes, a new pending IPRPC fund request is created on the Lava blockchain.
		  Use the rewards module's pending-ibc-iprpc-funds query to see the pending request. To apply the fund, 
		  use the rewards module's cover-ibc-iprpc-fund-cost transaction.
		  3. Use the relayer's query "channels" to get the channel ID of the IBC transfer channel. The IBC transfer 
		  channel's port ID is always "transfer"`,
		Args: cobra.ExactArgs(5),
		Example: `
		Submit the proposal:
		lavad tx rewards submit-ibc-iprpc-proposal <spec_id> <duration_in_months> <fund_amount> transfer <channel_id_with_lava> --from alice --node https://osmosis-rpc.publicnode.com:443
		
		Get the new proposal ID (might need to paginate):
		lavad q gov proposals --node https://osmosis-rpc.publicnode.com:443

		Send a deposit and vote:
		lavad tx gov deposit <proposal_id> 10000000ulava --from alice --node https://osmosis-rpc.publicnode.com:443
		lavad tx gov vote <proposal_id> yes --from alice --node https://osmosis-rpc.publicnode.com:443
		`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// get optional flags
			title, err := cmd.Flags().GetString(FlagTitle)
			if err != nil {
				return err
			}
			summary, err := cmd.Flags().GetString(FlagSummary)
			if err != nil {
				return err
			}
			expedited, err := cmd.Flags().GetBool(FlagExpedited)
			if err != nil {
				return err
			}

			// get args
			creator := clientCtx.FromAddress.String()
			gov, err := getGovModuleAddress(clientCtx)
			if err != nil {
				return err
			}

			amountStr := args[2]
			srcPort := args[3]
			srcChannel := args[4]
			coin, err := sdk.ParseCoinNormalized(amountStr)
			if err != nil {
				return err
			}
			amount := sdk.NewCoins(coin)

			// create transferGrantMsg TX
			transferGrantMsg := &authztypes.MsgGrant{
				Granter: creator,
				Grantee: gov,
				Grant:   authztypes.Grant{},
			}
			transferAuth := authztypes.NewGenericAuthorization(sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}))
			err = transferGrantMsg.SetAuthorization(transferAuth)
			if err != nil {
				return err
			}

			// create transferGrant TX
			revokeGrantMsg := &authztypes.MsgGrant{
				Granter: creator,
				Grantee: gov,
				Grant:   authztypes.Grant{},
			}
			revokeAuth := authztypes.NewGenericAuthorization(sdk.MsgTypeURL(&authztypes.MsgRevoke{}))
			err = revokeGrantMsg.SetAuthorization(revokeAuth)
			if err != nil {
				return err
			}

			// create unsigned TXs
			memoSpec := args[0]
			memoDuration := args[1]
			memo, err := createIbcIprpcMemo(creator, memoSpec, memoDuration)
			if err != nil {
				return err
			}
			timeoutHeight, err := getTimeoutHeight(clientCtx, srcPort, srcChannel)
			if err != nil {
				return err
			}
			transferMsg := ibctransfertypes.NewMsgTransfer(srcPort, srcChannel, amount[0], creator, types.IbcIprpcReceiverAddress().String(), timeoutHeight, 0, memo)
			transferRevokeMsg := authztypes.MsgRevoke{
				Granter:    creator,
				Grantee:    gov,
				MsgTypeUrl: sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}),
			}
			revokeRevokeMsg := authztypes.MsgRevoke{
				Granter:    creator,
				Grantee:    gov,
				MsgTypeUrl: sdk.MsgTypeURL(&authztypes.MsgRevoke{}),
			}

			// create authz Exec msg
			msgsAny, err := sdktx.SetMsgs([]sdk.Msg{transferMsg, &transferRevokeMsg, &revokeRevokeMsg})
			if err != nil {
				return err
			}
			execMsg := authztypes.MsgExec{
				Grantee: gov,
				Msgs:    msgsAny,
			}

			// create community spend msg
			communitySpendMsg := distributiontypes.MsgCommunityPoolSpend{
				Authority: gov,
				Recipient: creator,
				Amount:    amount,
			}

			// create submit proposal msg
			submitProposalMsg, err := govv1types.NewMsgSubmitProposal([]sdk.Msg{&communitySpendMsg, &execMsg}, sdk.NewCoins(), creator, "", title, summary, expedited)
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), revokeGrantMsg, transferGrantMsg, submitProposalMsg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	cmd.Flags().String(FlagTitle, "IPRPC over IBC proposal", "Proposal title")
	cmd.Flags().String(FlagSummary, "IPRPC over IBC proposal", "Proposal summary")
	cmd.Flags().Bool(FlagExpedited, false, "Expedited proposal")
	cmd.MarkFlagRequired(flags.FlagFrom)
	cmd.MarkFlagRequired(flags.FlagNode)
	return cmd
}

func getGovModuleAddress(clientCtx client.Context) (string, error) {
	authQuerier := authtypes.NewQueryClient(clientCtx)
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := authQuerier.ModuleAccountByName(timeoutCtx, &authtypes.QueryModuleAccountByNameRequest{Name: govtypes.ModuleName})
	if err != nil {
		return "", err
	}

	var govAcc authtypes.ModuleAccount
	err = clientCtx.Codec.Unmarshal(res.Account.Value, &govAcc)
	if err != nil {
		return "", err
	}

	return govAcc.Address, nil
}

// createIbcIprpcMemo creates the custom memo required for an ibc-transfer TX to trigger an IPRPC over IBC fund request
func createIbcIprpcMemo(creator string, spec string, durationStr string) (string, error) {
	duration, err := strconv.ParseUint(durationStr, 10, 64)
	if err != nil {
		return "", err
	}

	return types.CreateIprpcMemo(creator, spec, duration)
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
