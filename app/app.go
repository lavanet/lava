package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/x/fixationstore"
	fixationkeeper "github.com/lavanet/lava/v4/x/fixationstore/keeper"
	fixationtypes "github.com/lavanet/lava/v4/x/fixationstore/types"
	"github.com/lavanet/lava/v4/x/timerstore"
	timerstorekeeper "github.com/lavanet/lava/v4/x/timerstore/keeper"
	timerstoretypes "github.com/lavanet/lava/v4/x/timerstore/types"

	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	"github.com/cometbft/cometbft/libs/log"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/capability"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/evidence"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/cosmos/cosmos-sdk/x/feegrant/module"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	group "github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	groupmodule "github.com/cosmos/cosmos-sdk/x/group/module"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v7/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v7/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v7/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client"
	ibcclientclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/client"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcporttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v7/modules/light-clients/07-tendermint"
	"github.com/lavanet/lava/v4/app/keepers"
	appparams "github.com/lavanet/lava/v4/app/params"
	"github.com/lavanet/lava/v4/app/upgrades"
	"github.com/lavanet/lava/v4/docs"
	conflictmodule "github.com/lavanet/lava/v4/x/conflict"
	conflictmodulekeeper "github.com/lavanet/lava/v4/x/conflict/keeper"
	conflictmoduletypes "github.com/lavanet/lava/v4/x/conflict/types"
	downtimemodule "github.com/lavanet/lava/v4/x/downtime"
	downtimemodulekeeper "github.com/lavanet/lava/v4/x/downtime/keeper"
	downtimemoduletypes "github.com/lavanet/lava/v4/x/downtime/types"
	dualstakingmodule "github.com/lavanet/lava/v4/x/dualstaking"
	dualstakingmodulekeeper "github.com/lavanet/lava/v4/x/dualstaking/keeper"
	dualstakingmoduletypes "github.com/lavanet/lava/v4/x/dualstaking/types"
	epochstoragemodule "github.com/lavanet/lava/v4/x/epochstorage"
	epochstoragemodulekeeper "github.com/lavanet/lava/v4/x/epochstorage/keeper"
	epochstoragemoduletypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	pairingmodule "github.com/lavanet/lava/v4/x/pairing"
	pairingmoduleclient "github.com/lavanet/lava/v4/x/pairing/client"
	pairingmodulekeeper "github.com/lavanet/lava/v4/x/pairing/keeper"
	pairingmoduletypes "github.com/lavanet/lava/v4/x/pairing/types"
	plansmodule "github.com/lavanet/lava/v4/x/plans"
	plansmoduleclient "github.com/lavanet/lava/v4/x/plans/client"
	plansmodulekeeper "github.com/lavanet/lava/v4/x/plans/keeper"
	plansmoduletypes "github.com/lavanet/lava/v4/x/plans/types"
	projectsmodule "github.com/lavanet/lava/v4/x/projects"
	projectsmodulekeeper "github.com/lavanet/lava/v4/x/projects/keeper"
	projectsmoduletypes "github.com/lavanet/lava/v4/x/projects/types"
	protocolmodule "github.com/lavanet/lava/v4/x/protocol"
	protocolmoduleclient "github.com/lavanet/lava/v4/x/protocol/client/cli"
	protocolmodulekeeper "github.com/lavanet/lava/v4/x/protocol/keeper"
	protocolmoduletypes "github.com/lavanet/lava/v4/x/protocol/types"
	rewardsmodule "github.com/lavanet/lava/v4/x/rewards"
	rewardsmoduleclient "github.com/lavanet/lava/v4/x/rewards/client/cli"
	rewardsmodulekeeper "github.com/lavanet/lava/v4/x/rewards/keeper"
	rewardsmoduletypes "github.com/lavanet/lava/v4/x/rewards/types"
	specmodule "github.com/lavanet/lava/v4/x/spec"
	specmoduleclient "github.com/lavanet/lava/v4/x/spec/client"
	specmodulekeeper "github.com/lavanet/lava/v4/x/spec/keeper"
	specmoduletypes "github.com/lavanet/lava/v4/x/spec/types"
	subscriptionmodule "github.com/lavanet/lava/v4/x/subscription"
	subscriptionmodulekeeper "github.com/lavanet/lava/v4/x/subscription/keeper"
	subscriptionmoduletypes "github.com/lavanet/lava/v4/x/subscription/types"
	"github.com/spf13/cast"
	// this line is used by starport scaffolding # stargate/app/moduleImport
)

const (
	AccountAddressPrefix = "lava@"
	Name                 = "lava"
)

// Upgrades add here future upgrades (upgrades.Upgrade)
var Upgrades = []upgrades.Upgrade{
	upgrades.Upgrade_3_1_0,
	upgrades.Upgrade_4_0_0,
	upgrades.Upgrade_4_1_0,
}

// this line is used by starport scaffolding # stargate/wasm/app/enabledProposals

func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler
	// this line is used by starport scaffolding # stargate/app/govProposalHandlers

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		upgradeclient.LegacyProposalHandler,
		upgradeclient.LegacyCancelProposalHandler,
		ibcclientclient.UpdateClientProposalHandler,
		ibcclientclient.UpgradeProposalHandler,
		specmoduleclient.SpecAddProposalHandler,
		plansmoduleclient.PlansAddProposalHandler,
		plansmoduleclient.PlansDelProposalHandler,
		pairingmoduleclient.PairingUnstakeProposal,
		rewardsmoduleclient.SetIprpcDataProposalHandler,
		protocolmoduleclient.SetProtocolVersionProposalHandler,
		// this line is used by starport scaffolding # stargate/app/govProposalHandler
	)

	return govProposalHandlers
}

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		timerstore.AppModuleBasic{},
		fixationstore.AppModuleBasic{},
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(getGovProposalHandlers()),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		packetforward.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		transfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		specmodule.AppModuleBasic{},
		epochstoragemodule.AppModuleBasic{},
		dualstakingmodule.AppModuleBasic{},
		subscriptionmodule.AppModuleBasic{},
		pairingmodule.AppModuleBasic{},
		conflictmodule.AppModuleBasic{},
		projectsmodule.AppModuleBasic{},
		protocolmodule.AppModuleBasic{},
		plansmodule.AppModuleBasic{},
		downtimemodule.AppModuleBasic{},
		rewardsmodule.AppModuleBasic{},
		groupmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		// this line is used by starport scaffolding # stargate/app/moduleBasic
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:                                       nil,
		distrtypes.ModuleName:                                            nil,
		stakingtypes.BondedPoolName:                                      {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName:                                   {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:                                              {authtypes.Burner},
		ibctransfertypes.ModuleName:                                      {authtypes.Minter, authtypes.Burner},
		icatypes.ModuleName:                                              nil,
		subscriptionmoduletypes.ModuleName:                               {authtypes.Burner, authtypes.Staking},
		string(rewardsmoduletypes.ValidatorsRewardsAllocationPoolName):   {authtypes.Minter, authtypes.Staking},
		string(rewardsmoduletypes.ValidatorsRewardsDistributionPoolName): {authtypes.Burner, authtypes.Staking},
		string(rewardsmoduletypes.ValidatorsRewardsLeftOverPoolName):     {authtypes.Burner, authtypes.Staking},
		string(rewardsmoduletypes.ProviderRewardsDistributionPool):       {authtypes.Burner, authtypes.Staking},
		string(rewardsmoduletypes.ProvidersRewardsAllocationPool):        {authtypes.Minter, authtypes.Staking},
		dualstakingmoduletypes.ModuleName:                                {authtypes.Burner, authtypes.Staking},
		string(rewardsmoduletypes.IprpcPoolName):                         nil,
		// this line is used by starport scaffolding # stargate/app/maccPerms
	}
)

var (
	_ servertypes.Application = (*LavaApp)(nil)
	_ runtime.AppI            = (*LavaApp)(nil)
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, "."+Name)
}

// LavaApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type LavaApp struct {
	*baseapp.BaseApp
	// keepers
	keepers.LavaKeepers

	cdc               *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// this line is used by starport scaffolding # stargate/app/keeperDeclaration

	// mm is the module manager
	mm *module.Manager

	// sm is the simulation manager
	sm *module.SimulationManager

	// module configurator.
	configurator module.Configurator
}

// New returns a reference to an initialized blockchain app
func New(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig appparams.EncodingConfig,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *LavaApp {
	appCodec := encodingConfig.Marshaler
	cdc := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	level := appOpts.Get(flags.FlagLogLevel)
	utils.SetGlobalLoggingLevel(cast.ToString(level))

	bApp := baseapp.NewBaseApp(Name, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
		distrtypes.StoreKey, slashingtypes.StoreKey,
		govtypes.StoreKey, paramstypes.StoreKey, consensusparamtypes.StoreKey, upgradetypes.StoreKey, feegrant.StoreKey,
		evidencetypes.StoreKey, crisistypes.StoreKey, ibctransfertypes.StoreKey, ibcexported.StoreKey,
		icahosttypes.StoreKey, packetforwardtypes.StoreKey,
		icacontrollertypes.StoreKey,
		capabilitytypes.StoreKey,
		specmoduletypes.StoreKey,
		epochstoragemoduletypes.StoreKey,
		dualstakingmoduletypes.StoreKey,
		subscriptionmoduletypes.StoreKey,
		pairingmoduletypes.StoreKey,
		conflictmoduletypes.StoreKey,
		projectsmoduletypes.StoreKey,
		plansmoduletypes.StoreKey,
		downtimemoduletypes.StoreKey,
		rewardsmoduletypes.StoreKey,
		group.StoreKey,
		authzkeeper.StoreKey,
		// this line is used by starport scaffolding # stargate/app/storeKey
	)
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	memKeys := sdk.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	app := &LavaApp{
		BaseApp:           bApp,
		LavaKeepers:       keepers.LavaKeepers{},
		cdc:               cdc,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	app.ParamsKeeper = initParamsKeeper(appCodec, cdc, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

	// set the BaseApp's parameter store
	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(appCodec, keys[consensusparamtypes.StoreKey], authtypes.NewModuleAddress(govtypes.ModuleName).String())
	bApp.SetParamStore(&app.ConsensusParamsKeeper)

	// add capability keeper and ScopeToModule for ibc module
	app.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, keys[capabilitytypes.StoreKey], memKeys[capabilitytypes.MemStoreKey])

	// grant capabilities for the ibc and ibc-transfer modules
	scopedIBCKeeper := app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	scopedTransferKeeper := app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	scopedICAControllerKeeper := app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	scopedICAHostKeeper := app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	// this line is used by starport scaffolding # stargate/app/scopedKeeper

	// add keepers
	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec, keys[authtypes.StoreKey], authtypes.ProtoBaseAccount, maccPerms,
		sdk.GetConfig().GetBech32AccountAddrPrefix(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec, keys[banktypes.StoreKey], app.AccountKeeper, app.ModuleAccountAddrs(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec, keys[stakingtypes.StoreKey], app.AccountKeeper, app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec, keys[distrtypes.StoreKey], app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec, app.LegacyAmino(), keys[slashingtypes.StoreKey], app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.CrisisKeeper = *crisiskeeper.NewKeeper(
		appCodec,
		keys[crisistypes.StoreKey], invCheckPeriod, app.BankKeeper, authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(appCodec, keys[feegrant.StoreKey], app.AccountKeeper)
	app.GroupKeeper = groupkeeper.NewKeeper(keys[group.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper, group.DefaultConfig())
	app.AuthzKeeper = authzkeeper.NewKeeper(keys[authzkeeper.StoreKey], appCodec, app.MsgServiceRouter(), app.AccountKeeper)
	app.UpgradeKeeper = *upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Upgrade the KVStoreKey after upgrade keeper initialization
	app.setupUpgradeStoreLoaders()

	// ... other modules keepers

	// Create IBC Keeper
	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec, keys[ibcexported.StoreKey], app.GetSubspace(ibcexported.ModuleName), app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
	)

	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		appCodec,
		keys[packetforwardtypes.StoreKey],
		nil, // will be zero-value here, reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Initialize SpecKeeper prior to govRouter (order is critical)
	app.SpecKeeper = *specmodulekeeper.NewKeeper(
		appCodec,
		keys[specmoduletypes.StoreKey],
		keys[specmoduletypes.MemStoreKey],
		app.GetSubspace(specmoduletypes.ModuleName),
		app.StakingKeeper,
	)
	specModule := specmodule.NewAppModule(appCodec, app.SpecKeeper, app.AccountKeeper, app.BankKeeper)

	app.EpochstorageKeeper = *epochstoragemodulekeeper.NewKeeper(
		appCodec,
		keys[epochstoragemoduletypes.StoreKey],
		keys[epochstoragemoduletypes.MemStoreKey],
		app.GetSubspace(epochstoragemoduletypes.ModuleName),

		app.BankKeeper,
		app.AccountKeeper,
		app.SpecKeeper,
		app.StakingKeeper,
	)
	epochstorageModule := epochstoragemodule.NewAppModule(appCodec, app.EpochstorageKeeper, app.AccountKeeper, app.BankKeeper)

	// downtime module
	app.DowntimeKeeper = downtimemodulekeeper.NewKeeper(appCodec, keys[downtimemoduletypes.StoreKey], app.GetSubspace(downtimemoduletypes.ModuleName), app.EpochstorageKeeper)
	downtimeModule := downtimemodule.NewAppModule(app.DowntimeKeeper)

	// timerstore keeper
	app.TimerStoreKeeper = timerstorekeeper.NewKeeper(appCodec)

	// fixation store keeper
	app.FixationStoreKeeper = fixationkeeper.NewKeeper(appCodec, app.TimerStoreKeeper, app.EpochstorageKeeper.BlocksToSaveRaw)

	// Initialize PlansKeeper prior to govRouter (order is critical)
	app.PlansKeeper = *plansmodulekeeper.NewKeeper(
		appCodec,
		keys[plansmoduletypes.StoreKey],
		keys[plansmoduletypes.MemStoreKey],
		app.GetSubspace(plansmoduletypes.ModuleName),
		app.EpochstorageKeeper,
		app.SpecKeeper,
		app.FixationStoreKeeper,
		app.StakingKeeper,
	)
	plansModule := plansmodule.NewAppModule(appCodec, app.PlansKeeper)

	app.ProjectsKeeper = *projectsmodulekeeper.NewKeeper(
		appCodec,
		keys[projectsmoduletypes.StoreKey],
		keys[projectsmoduletypes.MemStoreKey],
		app.GetSubspace(projectsmoduletypes.ModuleName),
		app.EpochstorageKeeper,
		app.FixationStoreKeeper,
	)
	projectsModule := projectsmodule.NewAppModule(appCodec, app.ProjectsKeeper)

	app.DualstakingKeeper = *dualstakingmodulekeeper.NewKeeper(
		appCodec,
		keys[dualstakingmoduletypes.StoreKey],
		keys[dualstakingmoduletypes.MemStoreKey],
		app.GetSubspace(dualstakingmoduletypes.ModuleName),

		app.BankKeeper,
		app.StakingKeeper,
		app.AccountKeeper,
		app.EpochstorageKeeper,
		app.SpecKeeper,
		app.FixationStoreKeeper,
	)
	dualstakingModule := dualstakingmodule.NewAppModule(appCodec, app.DualstakingKeeper, app.AccountKeeper, app.BankKeeper)

	app.RewardsKeeper = *rewardsmodulekeeper.NewKeeper(
		appCodec,
		keys[rewardsmoduletypes.StoreKey],
		keys[rewardsmoduletypes.MemStoreKey],
		app.GetSubspace(rewardsmoduletypes.ModuleName),
		app.BankKeeper,
		app.AccountKeeper,
		app.SpecKeeper,
		app.EpochstorageKeeper,
		app.DowntimeKeeper,
		app.StakingKeeper,
		app.DualstakingKeeper,
		app.DistrKeeper,
		authtypes.FeeCollectorName,
		app.TimerStoreKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	rewardsModule := rewardsmodule.NewAppModule(appCodec, app.RewardsKeeper, app.AccountKeeper, app.BankKeeper)

	app.SubscriptionKeeper = *subscriptionmodulekeeper.NewKeeper(
		appCodec,
		keys[subscriptionmoduletypes.StoreKey],
		keys[subscriptionmoduletypes.MemStoreKey],
		app.GetSubspace(subscriptionmoduletypes.ModuleName),

		app.BankKeeper,
		app.AccountKeeper,
		&app.EpochstorageKeeper,
		app.ProjectsKeeper,
		app.PlansKeeper,
		app.DualstakingKeeper,
		app.RewardsKeeper,
		app.SpecKeeper,
		app.FixationStoreKeeper,
		app.TimerStoreKeeper,
		app.StakingKeeper,
	)
	subscriptionModule := subscriptionmodule.NewAppModule(appCodec, app.SubscriptionKeeper, app.AccountKeeper, app.BankKeeper)

	app.PairingKeeper = *pairingmodulekeeper.NewKeeper(
		appCodec,
		keys[pairingmoduletypes.StoreKey],
		keys[pairingmoduletypes.MemStoreKey],
		app.GetSubspace(pairingmoduletypes.ModuleName),

		app.BankKeeper,
		app.AccountKeeper,
		app.SpecKeeper,
		&app.EpochstorageKeeper,
		app.ProjectsKeeper,
		app.SubscriptionKeeper,
		app.PlansKeeper,
		app.DowntimeKeeper,
		app.DualstakingKeeper,
		app.StakingKeeper,
		app.FixationStoreKeeper,
		app.TimerStoreKeeper,
	)
	pairingModule := pairingmodule.NewAppModule(appCodec, app.PairingKeeper, app.AccountKeeper, app.BankKeeper)

	// register the proposal types
	govRouter := v1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, v1beta1.ProposalHandler).
		//
		// user defined
		AddRoute(specmoduletypes.ProposalsRouterKey, specmodule.NewSpecProposalsHandler(app.SpecKeeper)).
		// copied the code from param and changed the handler to enable functionality
		AddRoute(paramproposal.RouterKey, specmodule.NewParamChangeProposalHandler(app.ParamsKeeper)).
		// user defined
		AddRoute(plansmoduletypes.ProposalsRouterKey, plansmodule.NewPlansProposalsHandler(app.PlansKeeper)).
		AddRoute(pairingmoduletypes.ProposalsRouterKey, pairingmodule.NewPairingProposalsHandler(app.PairingKeeper)).

		//
		// default
		// TODO: Check alternatives to this
		// AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(app.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(&app.UpgradeKeeper)).
		// AddRoute(ibcexported.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper))
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper))

	// Create Transfer Keepers
	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.PacketForwardKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		scopedTransferKeeper,
	)
	transferModule := transfer.NewAppModule(app.TransferKeeper)
	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		&app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		scopedICAHostKeeper,
		app.MsgServiceRouter(),
	)
	icaControllerKeeper := icacontrollerkeeper.NewKeeper(
		appCodec, keys[icacontrollertypes.StoreKey],
		app.GetSubspace(icacontrollertypes.SubModuleName),
		app.IBCKeeper.ChannelKeeper, // may be replaced with middleware such as ics29 fee
		app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
		scopedICAControllerKeeper, app.MsgServiceRouter(),
	)
	icaModule := ica.NewAppModule(&icaControllerKeeper, &app.ICAHostKeeper)
	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)

	// Create evidence Keeper for to register the IBC light client misbehaviour evidence route
	evidenceKeeper := evidencekeeper.NewKeeper(
		appCodec, keys[evidencetypes.StoreKey], app.StakingKeeper, app.SlashingKeeper,
	)
	// If evidence needs to be handled for the app, set routes in router here and seal
	app.EvidenceKeeper = *evidenceKeeper

	govConfig := govtypes.Config{MaxMetadataLen: 3000} // 1640 TODO fix it from spec test proposal
	govKeeper := govkeeper.NewKeeper(
		appCodec, keys[govtypes.StoreKey], app.AccountKeeper, app.BankKeeper,
		app.StakingKeeper, app.MsgServiceRouter(), govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	govKeeper.SetLegacyRouter(govRouter)
	app.GovKeeper = *govKeeper.SetHooks(
		govtypes.NewMultiGovHooks(),
	)

	app.ConflictKeeper = *conflictmodulekeeper.NewKeeper(
		appCodec,
		keys[conflictmoduletypes.StoreKey],
		keys[conflictmoduletypes.MemStoreKey],
		app.GetSubspace(conflictmoduletypes.ModuleName),

		app.BankKeeper,
		app.AccountKeeper,
		app.PairingKeeper,
		app.EpochstorageKeeper,
		app.SpecKeeper,
		app.StakingKeeper,
	)
	conflictModule := conflictmodule.NewAppModule(appCodec, app.ConflictKeeper, app.AccountKeeper, app.BankKeeper)

	app.ProtocolKeeper = *protocolmodulekeeper.NewKeeper(
		appCodec,
		keys[protocolmoduletypes.StoreKey],
		keys[protocolmoduletypes.MemStoreKey],
		app.GetSubspace(protocolmoduletypes.ModuleName),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	protocolModule := protocolmodule.NewAppModule(appCodec, app.ProtocolKeeper)

	// this line is used by starport scaffolding # stargate/app/keeperDefinition

	// Create static IBC router, add transfer route, then set and seal it
	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)

	var transferStack ibcporttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = packetforward.NewIBCMiddleware(
		transferStack,
		app.PacketForwardKeeper,
		0, // retries on timeout
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp, // forward timeout
		packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,  // refund timeout
	)
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	// this line is used by starport scaffolding # ibc/app/router
	app.IBCKeeper.SetRouter(ibcRouter)

	/****  Module Options ****/

	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.

	// register the staking hooks
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks(), app.DualstakingKeeper.Hooks()),
	)

	app.mm = module.NewManager(
		genutil.NewAppModule(
			app.AccountKeeper, app.StakingKeeper, app.BaseApp.DeliverTx,
			encodingConfig.TxConfig,
		),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(appCodec, *app.CapabilityKeeper, false),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName)),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(&app.UpgradeKeeper),
		evidence.NewAppModule(app.EvidenceKeeper),
		groupmodule.NewAppModule(appCodec, app.GroupKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		authzmodule.NewAppModule(appCodec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		ibc.NewAppModule(app.IBCKeeper),
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		params.NewAppModule(app.ParamsKeeper),
		transferModule,
		timerstore.NewAppModule(app.TimerStoreKeeper),
		fixationstore.NewAppModule(app.FixationStoreKeeper),
		specModule,
		epochstorageModule,
		dualstakingModule,
		subscriptionModule,
		pairingModule,
		conflictModule,
		projectsModule,
		plansModule,
		protocolModule,
		downtimeModule,
		rewardsModule,
		icaModule,
		crisis.NewAppModule(&app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
		// this line is used by starport scaffolding # stargate/app/appModule
	)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	app.mm.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		timerstoretypes.ModuleName,
		fixationtypes.ModuleName,
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		rewardsmoduletypes.ModuleName, // rewards needs to run before distribution to send rewards to the fee collector
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		dualstakingmoduletypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		group.ModuleName,
		authz.ModuleName,
		icatypes.ModuleName,
		packetforwardtypes.ModuleName,
		specmoduletypes.ModuleName,
		epochstoragemoduletypes.ModuleName,
		subscriptionmoduletypes.ModuleName,
		conflictmoduletypes.ModuleName, // conflict needs to change state before pairing changes stakes
		downtimemoduletypes.ModuleName, // downtime needs to run before pairing
		pairingmoduletypes.ModuleName,
		projectsmoduletypes.ModuleName,
		plansmoduletypes.ModuleName,
		protocolmoduletypes.ModuleName,
		vestingtypes.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName)

	app.mm.SetOrderEndBlockers(
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		group.ModuleName,
		authz.ModuleName,
		icatypes.ModuleName,
		packetforwardtypes.ModuleName,
		specmoduletypes.ModuleName,
		epochstoragemoduletypes.ModuleName,
		dualstakingmoduletypes.ModuleName,
		subscriptionmoduletypes.ModuleName,
		conflictmoduletypes.ModuleName,
		pairingmoduletypes.ModuleName,
		projectsmoduletypes.ModuleName,
		protocolmoduletypes.ModuleName,
		plansmoduletypes.ModuleName,
		vestingtypes.ModuleName,
		rewardsmoduletypes.ModuleName,
		upgradetypes.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
		downtimemoduletypes.ModuleName, // downtime has no end block but module manager requires it.
		fixationtypes.ModuleName,       // fixation store has no end block but module manager requires it.
		timerstoretypes.ModuleName,     // timer store has no end block but module manager requires it.
	)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		epochstoragemoduletypes.ModuleName, // epochStyorage end block must come before pairing for proper epoch handling
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		dualstakingmoduletypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcexported.ModuleName,
		group.ModuleName,
		authz.ModuleName,
		icatypes.ModuleName,
		packetforwardtypes.ModuleName,
		specmoduletypes.ModuleName,
		subscriptionmoduletypes.ModuleName,
		downtimemoduletypes.ModuleName,
		pairingmoduletypes.ModuleName,
		projectsmoduletypes.ModuleName,
		plansmoduletypes.ModuleName,
		protocolmoduletypes.ModuleName,
		vestingtypes.ModuleName,
		upgradetypes.ModuleName,
		feegrant.ModuleName,
		rewardsmoduletypes.ModuleName,
		paramstypes.ModuleName,
		fixationtypes.ModuleName,       // fixation store has no init genesis but module manager requires it.
		timerstoretypes.ModuleName,     // timer store has no init genesis but module manager requires it.
		conflictmoduletypes.ModuleName, // NOTICE: the last module to initgenesis needs to push fixation in epoch storage
		// this line is used by starport scaffolding # stargate/app/initGenesis
	)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// setupUpgradeHandlers. must be called before LoadLatestVersion as storeloader is sealed after that
	app.setupUpgradeHandlers()

	// create the simulation manager and define the order of the modules for deterministic simulations
	app.sm = module.NewSimulationManager()

	// initialize stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)

	app.SetAnteHandler(
		NewAnteHandler(
			app.AccountKeeper,
			app.BankKeeper,
			app.DualstakingKeeper,
			encodingConfig.TxConfig.SignModeHandler(),
			app.FeeGrantKeeper,
			app.SpecKeeper,
			ante.DefaultSigVerificationGasConsumer),
	)
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}

	app.ScopedIBCKeeper = scopedIBCKeeper
	app.ScopedTransferKeeper = scopedTransferKeeper
	// this line is used by starport scaffolding # stargate/app/beforeInitReturn

	return app
}

// setupUpgradeStoreLoaders when intoducing new modules.
func (app *LavaApp) setupUpgradeStoreLoaders() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	for _, upgrade := range Upgrades {
		if upgradeInfo.Name == upgrade.UpgradeName {
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &upgrade.StoreUpgrades))
		}
	}
}

// setupUpgradeHandlers when modifing already existing modules
func (app *LavaApp) setupUpgradeHandlers() {
	for _, upgrade := range Upgrades {
		app.UpgradeKeeper.SetUpgradeHandler(
			upgrade.UpgradeName,
			upgrade.CreateUpgradeHandler(
				app.mm,
				app.configurator,
				app.BaseApp,
				&app.LavaKeepers,
			),
		)
	}
}

// Name returns the name of the App
func (app *LavaApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *LavaApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *LavaApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *LavaApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())
	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *LavaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

func (app *LavaApp) RegisterNodeService(clientCtx client.Context) {
	node.RegisterNodeService(clientCtx, app.GRPCQueryRouter())
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *LavaApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// LegacyAmino returns SimApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *LavaApp) LegacyAmino() *codec.LegacyAmino {
	return app.cdc
}

// AppCodec returns an app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *LavaApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns an InterfaceRegistry
func (app *LavaApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *LavaApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *LavaApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tkeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *LavaApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *LavaApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *LavaApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register node gRPC service for grpc-gateway.
	node.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register legacy and grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register app's OpenAPI routes.
	docs.RegisterOpenAPIService(Name, apiSvr.Router)
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *LavaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *LavaApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// GetMaccPerms returns a copy of the module account permissions
func GetMaccPerms() map[string][]string {
	dupMaccPerms := make(map[string][]string)
	for k, v := range maccPerms {
		dupMaccPerms[k] = v
	}
	return dupMaccPerms
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(v1.ParamKeyTable()) //nolint:staticcheck
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(packetforwardtypes.ModuleName).WithKeyTable(packetforwardtypes.ParamKeyTable())
	paramsKeeper.Subspace(specmoduletypes.ModuleName)
	paramsKeeper.Subspace(epochstoragemoduletypes.ModuleName)
	paramsKeeper.Subspace(dualstakingmoduletypes.ModuleName)
	paramsKeeper.Subspace(subscriptionmoduletypes.ModuleName)
	paramsKeeper.Subspace(pairingmoduletypes.ModuleName)
	paramsKeeper.Subspace(conflictmoduletypes.ModuleName)
	paramsKeeper.Subspace(projectsmoduletypes.ModuleName)
	paramsKeeper.Subspace(protocolmoduletypes.ModuleName)
	paramsKeeper.Subspace(plansmoduletypes.ModuleName)
	paramsKeeper.Subspace(downtimemoduletypes.ModuleName)
	paramsKeeper.Subspace(rewardsmoduletypes.ModuleName)
	paramsKeeper.Subspace(group.ModuleName)
	paramsKeeper.Subspace(authz.ModuleName)
	// this line is used by starport scaffolding # stargate/app/paramSubspace

	return paramsKeeper
}

// SimulationManager implements the SimulationApp interface
func (app *LavaApp) SimulationManager() *module.SimulationManager {
	return app.sm
}
