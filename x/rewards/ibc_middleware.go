package rewards

import (
	"errors"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the transfer middleware given
// the rewards keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.IBCModule // transfer stack
	keeper keeper.Keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application
func NewIBCMiddleware(app porttypes.IBCModule, k keeper.Keeper) IBCMiddleware {
	return IBCMiddleware{
		app:    app,
		keeper: k,
	}
}

// IBCModule interface implementation. Only OnRecvPacket() calls a callback from the keeper. The rest have default implementations

func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (version string, err error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID, channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket checks the packet's memo and funds the IPRPC pool accordingly. If the memo is not the expected JSON,
// the packet is transferred normally to the next IBC module in the transfer stack
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	// unmarshal the packet's data with the transfer module codec (expect an ibc-transfer packet)
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// sanity check: validate data
	if err := data.ValidateBasic(); err != nil {
		detaliedErr := utils.LavaFormatError("rewards module IBC middleware processing failed", err,
			utils.LogAttr("data", data),
		)
		return channeltypes.NewErrorAcknowledgement(detaliedErr)
	}

	// extract the packet's memo
	memo, err := im.keeper.ExtractIprpcMemoFromPacket(ctx, data)
	if errors.Is(err, types.ErrMemoNotIprpcOverIbc) {
		// not a packet that should be handled as IPRPC over IBC (not considered as error)
		utils.LavaFormatDebug("rewards module IBC middleware processing skipped, memo is invalid for IPRPC over IBC funding",
			utils.LogAttr("memo", memo),
		)
		return im.app.OnRecvPacket(ctx, packet, relayer)
	} else if errors.Is(err, types.ErrIprpcMemoInvalid) {
		// memo is in the right format of IPRPC over IBC but the data is invalid
		utils.LavaFormatWarning("rewards module IBC middleware processing failed, memo data is invalid", err,
			utils.LogAttr("memo", memo))
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// create the IBC tokens to register it in the pending IPRPC over IBC object
	amount, ok := sdk.NewIntFromString(data.Amount)
	if !ok {
		detaliedErr := utils.LavaFormatError("rewards module IBC middleware processing failed", fmt.Errorf("invalid amount in ibc-transfer data"),
			utils.LogAttr("data", data),
		)
		return channeltypes.NewErrorAcknowledgement(detaliedErr)
	}
	ibcTokens := transfertypes.GetTransferCoin(packet.DestinationPort, packet.DestinationChannel, data.Denom, amount)

	// set pending IPRPC over IBC requests on-chain
	// leftovers are the tokens that were not registered in the PendingIbcIprpcFund objects due to int division
	piif, leftovers, err := im.keeper.NewPendingIbcIprpcFund(ctx, memo.Creator, memo.Spec, memo.Duration, ibcTokens)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// send the IBC transfer to get the tokens
	ack := im.sendIbcTransfer(ctx, packet, relayer, data.Sender, types.IbcIprpcReceiverAddress().String())
	if ack == nil || !ack.Success() {
		// we check for ack == nil because it means that IBC transfer module did not return an acknowledgement.
		// This isn't necessarily an error, but it could indicate unexpected behavior or asynchronous processing
		// on the IBC transfer module's side (which returns a non-nil ack when executed without errors). Asynchronous
		// processing can be queued processing of packets, interacting with external APIs and more. These can cause
		// delays in the IBC-transfer's processing which will make the module return a nil ack until the processing is done.
		return ack
	}

	// transfer the token from the temp IbcIprpcReceiverAddress to the pending IPRPC fund request pool
	// we subtract the leftovers since they are not registered in the PendingIbcIprpcFund store
	err1 := im.keeper.SendIbcTokensToPendingIprpcPool(ctx, ibcTokens.Sub(leftovers))
	if err1 != nil {
		// we couldn't transfer the funds to the pending IPRPC fund request pool, try moving it to the community pool
		err2 := im.keeper.FundCommunityPoolFromIbcIprpcReceiver(ctx, ibcTokens)
		if err2 != nil {
			// community pool transfer failed, token kept locked in IbcIprpcReceiverAddress, return err ack
			err := utils.LavaFormatError("could not send tokens from IbcIprpcReceiverAddress to pending IPRPC pool or community pool, tokens are locked in IbcIprpcReceiverAddress",
				errors.Join(err1, err2),
				utils.LogAttr("amount", ibcTokens.String()),
				utils.LogAttr("sender", data.Sender),
			)
			return channeltypes.NewErrorAcknowledgement(err)
		} else {
			err := utils.LavaFormatError("could not send tokens from IbcIprpcReceiverAddress to pending IPRPC pool, sent to community pool", err1,
				utils.LogAttr("amount", ibcTokens.String()),
				utils.LogAttr("sender", data.Sender),
			)
			return channeltypes.NewErrorAcknowledgement(err)
		}
	}

	// transfer the leftover IBC tokens to the community pool
	err = im.keeper.FundCommunityPoolFromIbcIprpcReceiver(ctx, leftovers)
	if err != nil {
		detailedErr := utils.LavaFormatError("could not send leftover tokens from IbcIprpcReceiverAddress to community pool, tokens are locked in IbcIprpcReceiverAddress", err,
			utils.LogAttr("amount", leftovers.String()),
			utils.LogAttr("sender", data.Sender),
		)
		return channeltypes.NewErrorAcknowledgement(detailedErr)
	}

	// make event
	utils.LogLavaEvent(ctx, im.keeper.Logger(ctx), types.NewPendingIbcIprpcFundEventName, map[string]string{
		"index":        strconv.FormatUint(piif.Index, 10),
		"creator":      piif.Creator,
		"spec":         piif.Spec,
		"duration":     strconv.FormatUint(piif.Duration, 10),
		"monthly_fund": piif.Fund.String(),
		"expiry":       strconv.FormatUint(piif.Expiry, 10),
	}, "New pending IBC IPRPC fund was created successfully")

	return channeltypes.NewResultAcknowledgement([]byte{byte(1)})
}

func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// ICS4Wrapper interface (default implementations)
func (im IBCMiddleware) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort string, sourceChannel string,
	timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte,
) (sequence uint64, err error) {
	return im.keeper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.PacketI,
	ack exported.Acknowledgement,
) error {
	return im.keeper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// sendIbcTransfer sends the ibc-transfer packet to the IBC Transfer module
func (im IBCMiddleware) sendIbcTransfer(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, sender string, receiver string) exported.Acknowledgement {
	// unmarshal the packet's data with the transfer module codec (expect an ibc-transfer packet)
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// change the ibc-transfer packet data with sender and receiver
	data.Sender = sender
	data.Receiver = receiver
	data.Memo = ""
	marshelledData, err := transfertypes.ModuleCdc.MarshalJSON(&data)
	if err != nil {
		utils.LavaFormatError("rewards module IBC middleware processing failed, cannot marshal packet data", err,
			utils.LogAttr("data", data))
		return channeltypes.NewErrorAcknowledgement(err)
	}
	packet.Data = marshelledData

	// call the next OnRecvPacket() of the transfer stack to make the IBC Transfer module's OnRecvPacket send the IBC tokens
	return im.app.OnRecvPacket(ctx, packet, relayer)
}
