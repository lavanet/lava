package keepers

import (
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	conflictmodulekeeper "github.com/lavanet/lava/x/conflict/keeper"
	downtimemodulekeeper "github.com/lavanet/lava/x/downtime/keeper"
	dualstakingmodulekeeper "github.com/lavanet/lava/x/dualstaking/keeper"
	epochstoragemodulekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	pairingmodulekeeper "github.com/lavanet/lava/x/pairing/keeper"
	plansmodulekeeper "github.com/lavanet/lava/x/plans/keeper"
	projectsmodulekeeper "github.com/lavanet/lava/x/projects/keeper"
	protocolmodulekeeper "github.com/lavanet/lava/x/protocol/keeper"
	specmodulekeeper "github.com/lavanet/lava/x/spec/keeper"
	subscriptionmodulekeeper "github.com/lavanet/lava/x/subscription/keeper"
	// this line is used by starport scaffolding # stargate/app/moduleImport
)

type LavaKeepers struct {
	// Standard Keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    *stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	MintKeeper       mintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	CrisisKeeper     crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	IBCKeeper        *ibckeeper.Keeper // IBC Keeper must be a pointer in the lk, so we can SetRouter on it correctly
	EvidenceKeeper   evidencekeeper.Keeper
	TransferKeeper   ibctransferkeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper

	// Special Keepers
	SpecKeeper         specmodulekeeper.Keeper
	SubscriptionKeeper subscriptionmodulekeeper.Keeper
	EpochstorageKeeper epochstoragemodulekeeper.Keeper
	DualstakingKeeper  dualstakingmodulekeeper.Keeper
	PairingKeeper      pairingmodulekeeper.Keeper
	ConflictKeeper     conflictmodulekeeper.Keeper
	ProjectsKeeper     projectsmodulekeeper.Keeper
	PlansKeeper        plansmodulekeeper.Keeper
	ProtocolKeeper     protocolmodulekeeper.Keeper
	DowntimeKeeper     downtimemodulekeeper.Keeper

	ConsensusParamsKeeper consensusparamkeeper.Keeper
}
