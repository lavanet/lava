package app

import (
	txsigning "cosmossdk.io/x/tx/signing"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	dualstakingante "github.com/lavanet/lava/v4/x/dualstaking/ante"
	dualstakingkeeper "github.com/lavanet/lava/v4/x/dualstaking/keeper"
	specante "github.com/lavanet/lava/v4/x/spec/ante"
	"github.com/lavanet/lava/v4/x/spec/keeper"
)

func NewAnteHandler(accountKeeper ante.AccountKeeper, bankKeeper authtypes.BankKeeper, dualstakingKeeper dualstakingkeeper.Keeper, signModeHandler *txsigning.HandlerMap, feegrantKeeper ante.FeegrantKeeper, specKeeper keeper.Keeper, sigGasConsumer ante.SignatureVerificationGasConsumer) sdk.AnteHandler {
	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(accountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(accountKeeper),
		ante.NewDeductFeeDecorator(accountKeeper, bankKeeper, feegrantKeeper, nil),
		ante.NewSetPubKeyDecorator(accountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(accountKeeper),
		ante.NewSigGasConsumeDecorator(accountKeeper, sigGasConsumer),
		ante.NewSigVerificationDecorator(accountKeeper, signModeHandler),
		ante.NewIncrementSequenceDecorator(accountKeeper),
		dualstakingante.NewRedelegationFlager(dualstakingKeeper),
		specante.NewExpeditedProposalFilterAnteDecorator(specKeeper),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}
