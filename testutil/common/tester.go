package common

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	"github.com/lavanet/lava/v2/utils/sigs"
	dualstakingante "github.com/lavanet/lava/v2/x/dualstaking/ante"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationstoretypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/v2/x/subscription/types"
	"github.com/stretchr/testify/require"
)

type Tester struct {
	T *testing.T

	GoCtx   context.Context
	Ctx     sdk.Context
	Servers *testkeeper.Servers
	Keepers *testkeeper.Keepers

	accounts map[string]sigs.Account
	plans    map[string]planstypes.Plan
	policies map[string]planstypes.Policy
	projects map[string]projectstypes.ProjectData
	specs    map[string]spectypes.Spec
}

const (
	PROVIDER  string = "provider_"
	CONSUMER  string = "consumer_"
	VALIDATOR string = "validator_"
	DEVELOPER string = "developer_"
)

func NewTester(t *testing.T) *Tester {
	ts := NewTesterRaw(t)

	// AdvanceBlock() and AdvanceEpoch() always use the current time for the
	// first block (and ignores the time delta arg if given); So call it here
	// to generate a first timestamp and avoid any subsequent call with delta
	// argument call having the delta ignored.
	ts.AdvanceEpoch()

	// On the 28th above day of the month, some tests fail because NextMonth is always truncating the day
	// back to the 28th, so some timers will not trigger after executing ts.AdvanceMonths(1)
	if ts.Ctx.BlockTime().Day() >= 26 {
		ts.AdvanceBlock(5 * 24 * time.Hour)
	}

	return ts
}

func NewTesterRaw(t *testing.T) *Tester {
	servers, keepers, GoCtx := testkeeper.InitAllKeepers(t)

	ts := &Tester{
		T:       t,
		GoCtx:   GoCtx,
		Ctx:     sdk.UnwrapSDKContext(GoCtx),
		Servers: servers,
		Keepers: keepers,

		accounts: make(map[string]sigs.Account),
		plans:    make(map[string]planstypes.Plan),
		policies: make(map[string]planstypes.Policy),
		projects: make(map[string]projectstypes.ProjectData),
		specs:    make(map[string]spectypes.Spec),
	}

	return ts
}

func (ts *Tester) SetChainID(chainID string) {
	blockHeader := ts.Ctx.BlockHeader()
	blockHeader.ChainID = chainID
	ts.Ctx = ts.Ctx.WithBlockHeader(blockHeader)
	ts.GoCtx = sdk.WrapSDKContext(ts.Ctx)
}

func (ts *Tester) SetupAccounts(numSub, numAdm, numDev int) *Tester {
	for i := 1; i <= numSub; i++ {
		ts.AddAccount("sub", i, 20000)
	}
	for i := 1; i <= numAdm; i++ {
		ts.AddAccount("adm", i, 10000)
	}
	for i := 1; i <= numDev; i++ {
		ts.AddAccount("dev", i, 10000)
	}

	return ts
}

func (ts *Tester) AddAccount(kind string, idx int, balance int64) (sigs.Account, string) {
	name := kind + strconv.Itoa(idx)
	acc := CreateNewAccount(ts.GoCtx, *ts.Keepers, balance)
	if kind == PROVIDER {
		vault := CreateNewAccount(ts.GoCtx, *ts.Keepers, balance)
		acc.Vault = &vault
	}
	ts.accounts[name] = acc
	return ts.Account(name)
}

func (ts *Tester) GetAccount(kind string, idx int) (sigs.Account, string) {
	name := kind + strconv.Itoa(idx)
	return ts.Account(name)
}

func (ts *Tester) Account(name string) (sigs.Account, string) {
	account, ok := ts.accounts[name]
	if !ok {
		panic("tester: unknown account name: '" + name + "'")
	}
	return account, account.Addr.String()
}

func (ts *Tester) Accounts(name string) []sigs.Account {
	var names []string
	for k := range ts.accounts {
		if strings.HasPrefix(k, name) {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	var accounts []sigs.Account
	for _, k := range names {
		accounts = append(accounts, ts.accounts[k])
	}
	return accounts
}

func (ts *Tester) AccountsMap() map[string]sigs.Account {
	return ts.accounts
}

func (ts *Tester) StakeProvider(vault string, provider string, spec spectypes.Spec, amount int64) error {
	d := MockDescription()
	return ts.StakeProviderExtra(vault, provider, spec, amount, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
}

func (ts *Tester) StakeProviderExtra(
	vault string,
	provider string,
	spec spectypes.Spec,
	amount int64,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
	identity string,
	website string,
	securityContact string,
	descriptionDetails string,
) error {
	// if geoloc left zero, use default 1
	if geoloc == 0 {
		geoloc = 1
	}

	// if necessary, generate mock endpoints
	if endpoints == nil {
		apiInterfaces := []string{}
		for _, apiCollection := range spec.ApiCollections {
			apiInterfaces = append(apiInterfaces, apiCollection.CollectionData.ApiInterface)
		}
		geolocations := planstypes.GetGeolocationsFromUint(geoloc)

		for _, geo := range geolocations {
			endpoint := epochstoragetypes.Endpoint{
				IPPORT:        "123",
				ApiInterfaces: apiInterfaces,
				Geolocation:   int32(geo),
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	stake := sdk.NewCoin(ts.TokenDenom(), sdk.NewInt(amount))
	description := stakingtypes.NewDescription(moniker, identity, website, securityContact, descriptionDetails)
	_, err := ts.TxPairingStakeProvider(vault, provider, spec.Index, stake, endpoints, geoloc, description)

	return err
}

// GetValidator gets a validator object
// Usually, you get the account of your created validator with ts.GetAccount
// so input valAcc.addr to this function
func (ts *Tester) GetValidator(addr sdk.AccAddress) stakingtypes.Validator {
	v, found := ts.Keepers.StakingKeeper.GetValidator(ts.Ctx, sdk.ValAddress(addr))
	require.True(ts.T, found)
	return v
}

// SlashValidator slashes a validator and returns the expected amount of burned tokens (of the validator).
// Usually, you get the account of your created validator with ts.GetAccount, so input valAcc to this function
func (ts *Tester) SlashValidator(valAcc sigs.Account, fraction math.LegacyDec, power int64, block int64) math.Int {
	// slash
	valConsAddr := sdk.GetConsAddress(valAcc.PubKey)
	ts.Keepers.SlashingKeeper.Slash(ts.Ctx, valConsAddr, fraction, power, ts.Ctx.BlockHeight())

	var req abci.RequestBeginBlock
	ts.Keepers.Dualstaking.BeginBlock(ts.Ctx, req)

	// calculate expected burned tokens
	consensusPowerTokens := ts.Keepers.StakingKeeper.TokensFromConsensusPower(ts.Ctx, power)
	return fraction.MulInt(consensusPowerTokens).TruncateInt()
}

func (ts *Tester) AccountByAddr(addr string) (sigs.Account, string) {
	for _, account := range ts.accounts {
		if account.Addr.String() == addr {
			return account, addr
		}
	}
	panic("tester: unknown account address: '" + addr + "'")
}

func (ts *Tester) AddPlan(name string, plan planstypes.Plan) *Tester {
	err := ts.Keepers.Plans.AddPlan(ts.Ctx, plan, false)
	if err != nil {
		panic("tester: falied to add plan: '" + plan.Index + "'")
	}
	ts.plans[name] = plan
	return ts
}

func (ts *Tester) ModifyPlan(name string, plan planstypes.Plan) *Tester {
	err := ts.Keepers.Plans.AddPlan(ts.Ctx, plan, false)
	if err != nil {
		panic("tester: falied to add plan: '" + plan.Index + "'")
	}
	ts.plans[name] = plan
	return ts
}

func (ts *Tester) Plan(name string) planstypes.Plan {
	plan, ok := ts.plans[name]
	if !ok {
		panic("tester: unknown plan: '" + name + "'")
	}
	return plan
}

func (ts *Tester) AddPolicy(name string, policy planstypes.Policy) *Tester {
	ts.policies[name] = policy
	return ts
}

func (ts *Tester) Policy(name string) planstypes.Policy {
	policy, ok := ts.policies[name]
	if !ok {
		panic("tester: unknown policy: '" + name + "'")
	}
	return policy
}

func (ts *Tester) TokenDenom() string {
	return ts.Keepers.StakingKeeper.BondDenom(ts.Ctx)
}

func (ts *Tester) AddProjectData(name string, pd projectstypes.ProjectData) *Tester {
	ts.projects[name] = pd
	return ts
}

func (ts *Tester) ProjectData(name string) projectstypes.ProjectData {
	project, ok := ts.projects[name]
	if !ok {
		panic("tester: unknown project: '" + name + "'")
	}
	return project
}

func (ts *Tester) AddSpec(name string, spec spectypes.Spec) *Tester {
	ts.Keepers.Spec.SetSpec(ts.Ctx, spec)
	ts.specs[name] = spec
	return ts
}

func (ts *Tester) Spec(name string) spectypes.Spec {
	spec, ok := ts.specs[name]
	if !ok {
		panic("tester: unknown spec: '" + name + "'")
	}
	return spec
}

// misc shortcuts

func NewCoin(tokenDenom string, amount int64) sdk.Coin {
	return sdk.NewCoin(tokenDenom, sdk.NewInt(amount))
}

func NewCoins(tokenDenom string, amount ...int64) []sdk.Coin {
	return lavaslices.Map(amount, func(a int64) sdk.Coin { return NewCoin(tokenDenom, a) })
}

// keeper helpers

func (ts *Tester) GetBalance(accAddr sdk.AccAddress) int64 {
	denom := ts.Keepers.StakingKeeper.BondDenom(ts.Ctx)
	return ts.Keepers.BankKeeper.GetBalance(ts.Ctx, accAddr, denom).Amount.Int64()
}

func (ts *Tester) GetBalances(accAddr sdk.AccAddress) sdk.Coins {
	return ts.Keepers.BankKeeper.GetAllBalances(ts.Ctx, accAddr)
}

func (ts *Tester) FindPlan(index string, block uint64) (planstypes.Plan, bool) {
	return ts.Keepers.Plans.FindPlan(ts.Ctx, index, block)
}

func (ts *Tester) GetPlanFromSubscription(addr string, block uint64) (planstypes.Plan, error) {
	return ts.Keepers.Subscription.GetPlanFromSubscription(ts.Ctx, addr, block)
}

func (ts *Tester) GetProjectForBlock(projectID string, block uint64) (projectstypes.Project, error) {
	return ts.Keepers.Projects.GetProjectForBlock(ts.Ctx, projectID, block)
}

func (ts *Tester) GetProjectForDeveloper(devkey string, block uint64) (projectstypes.Project, error) {
	return ts.Keepers.Projects.GetProjectForDeveloper(ts.Ctx, devkey, block)
}

func (ts *Tester) GetProjectDeveloperData(devkey string, block uint64) (projectstypes.ProtoDeveloperData, error) {
	return ts.Keepers.Projects.GetProjectDeveloperData(ts.Ctx, devkey, block)
}

func (ts *Tester) VotePeriod() uint64 {
	return ts.Keepers.Conflict.VotePeriod(ts.Ctx)
}

func (ts *Tester) ChangeDelegationTimestamp(provider, delegator, chainID string, block uint64, timestamp int64) error {
	index := dualstakingtypes.DelegationKey(provider, delegator, chainID)
	return ts.Keepers.Dualstaking.ChangeDelegationTimestampForTesting(ts.Ctx, index, block, timestamp)
}

// proposals, transactions, queries

func (ts *Tester) TxProposalChangeParam(module, paramKey, paramVal string) error {
	return testkeeper.SimulateParamChange(ts.Ctx, ts.Keepers.ParamsKeeper, module, paramKey, paramVal)
}

func (ts *Tester) TxProposalAddPlans(plans ...planstypes.Plan) error {
	return testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans, false)
}

func (ts *Tester) TxProposalDelPlans(indices ...string) error {
	return testkeeper.SimulatePlansDelProposal(ts.Ctx, ts.Keepers.Plans, indices)
}

func (ts *Tester) TxProposalAddSpecs(specs ...spectypes.Spec) error {
	return testkeeper.SimulateSpecAddProposal(ts.Ctx, ts.Keepers.Spec, specs)
}

// TxDualstakingDelegate: implement 'tx dualstaking delegate'
func (ts *Tester) TxDualstakingDelegate(
	creator string,
	provider string,
	chainID string,
	amount sdk.Coin,
) (*dualstakingtypes.MsgDelegateResponse, error) {
	validator, _ := ts.GetAccount(VALIDATOR, 0)
	return ts.TxDualstakingDelegateValidator(
		creator,
		sdk.ValAddress(validator.Addr).String(),
		provider,
		chainID,
		amount,
	)
}

// TxDualstakingDelegate: implement 'tx dualstaking delegate'
func (ts *Tester) TxDualstakingDelegateValidator(
	creator string,
	validator string,
	provider string,
	chainID string,
	amount sdk.Coin,
) (*dualstakingtypes.MsgDelegateResponse, error) {
	msg := &dualstakingtypes.MsgDelegate{
		Creator:   creator,
		Validator: validator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    amount,
	}
	return ts.Servers.DualstakingServer.Delegate(ts.GoCtx, msg)
}

// TxDualstakingDelegate: implement 'tx dualstaking delegate'
func (ts *Tester) TxDualstakingRedelegate(
	creator string,
	fromProvider string,
	toProvider string,
	fromChainID string,
	toChainID string,
	amount sdk.Coin,
) (*dualstakingtypes.MsgRedelegateResponse, error) {
	msg := &dualstakingtypes.MsgRedelegate{
		Creator:      creator,
		FromProvider: fromProvider,
		ToProvider:   toProvider,
		FromChainID:  fromChainID,
		ToChainID:    toChainID,
		Amount:       amount,
	}
	return ts.Servers.DualstakingServer.Redelegate(ts.GoCtx, msg)
}

// TxDualstakingUnbond: implement 'tx dualstaking unbond'
func (ts *Tester) TxDualstakingUnbond(
	creator string,
	provider string,
	chainID string,
	amount sdk.Coin,
) (*dualstakingtypes.MsgUnbondResponse, error) {
	validator, _ := ts.GetAccount(VALIDATOR, 0)
	return ts.TxDualstakingUnbondValidator(
		creator,
		sdk.ValAddress(validator.Addr).String(),
		provider,
		chainID,
		amount,
	)
}

// TxDualstakingUnbond: implement 'tx dualstaking unbond'
func (ts *Tester) TxDualstakingUnbondValidator(
	creator string,
	validator string,
	provider string,
	chainID string,
	amount sdk.Coin,
) (*dualstakingtypes.MsgUnbondResponse, error) {
	msg := &dualstakingtypes.MsgUnbond{
		Creator:   creator,
		Validator: validator,
		Provider:  provider,
		ChainID:   chainID,
		Amount:    amount,
	}
	return ts.Servers.DualstakingServer.Unbond(ts.GoCtx, msg)
}

// TxDualstakingClaimRewards: implement 'tx dualstaking claim-rewards'
func (ts *Tester) TxDualstakingClaimRewards(
	creator string,
	provider string,
) (*dualstakingtypes.MsgClaimRewardsResponse, error) {
	msg := &dualstakingtypes.MsgClaimRewards{
		Creator:  creator,
		Provider: provider,
	}
	return ts.Servers.DualstakingServer.ClaimRewards(ts.GoCtx, msg)
}

// TxSubscriptionBuy: implement 'tx subscription buy'
func (ts *Tester) TxSubscriptionBuy(creator, consumer, plan string, months int, autoRenewal, advancePurchase bool) (*subscriptiontypes.MsgBuyResponse, error) {
	msg := &subscriptiontypes.MsgBuy{
		Creator:         creator,
		Consumer:        consumer,
		Index:           plan,
		Duration:        uint64(months),
		AutoRenewal:     autoRenewal,
		AdvancePurchase: advancePurchase,
	}
	return ts.Servers.SubscriptionServer.Buy(ts.GoCtx, msg)
}

// TxSubscriptionAddProject: implement 'tx subscription add-project'
func (ts *Tester) TxSubscriptionAddProject(creator string, pd projectstypes.ProjectData) error {
	msg := &subscriptiontypes.MsgAddProject{
		Creator:     creator,
		ProjectData: pd,
	}
	_, err := ts.Servers.SubscriptionServer.AddProject(ts.GoCtx, msg)
	return err
}

// TxSubscriptionDelProject: implement 'tx subscription del-project'
func (ts *Tester) TxSubscriptionDelProject(creator, projectName string) error {
	msg := &subscriptiontypes.MsgDelProject{
		Creator: creator,
		Name:    projectName,
	}
	_, err := ts.Servers.SubscriptionServer.DelProject(ts.GoCtx, msg)
	return err
}

// TxSubscriptionAutoRenewal: implement 'tx subscription auto-renewal'
func (ts *Tester) TxSubscriptionAutoRenewal(creator, consumer, planIndex string, enable bool) error {
	msg := &subscriptiontypes.MsgAutoRenewal{
		Creator:  creator,
		Consumer: consumer,
		Enable:   enable,
		Index:    planIndex,
	}
	_, err := ts.Servers.SubscriptionServer.AutoRenewal(ts.GoCtx, msg)
	return err
}

// TxProjectAddKeys: implement 'tx project add-keys'
func (ts *Tester) TxProjectAddKeys(projectID, creator string, projectKeys ...projectstypes.ProjectKey) error {
	msg := projectstypes.MsgAddKeys{
		Creator:     creator,
		Project:     projectID,
		ProjectKeys: projectKeys,
	}
	_, err := ts.Servers.ProjectServer.AddKeys(ts.GoCtx, &msg)
	return err
}

// TxProjectDelKeys: implement 'tx project del-keys'
func (ts *Tester) TxProjectDelKeys(projectID, creator string, projectKeys ...projectstypes.ProjectKey) error {
	msg := projectstypes.MsgDelKeys{
		Creator:     creator,
		Project:     projectID,
		ProjectKeys: projectKeys,
	}
	_, err := ts.Servers.ProjectServer.DelKeys(ts.GoCtx, &msg)
	return err
}

// TxProjectSetSubscriptionPolicy: implement 'tx project set-subscription-policy'
func (ts *Tester) TxProjectSetSubscriptionPolicy(projectID, subkey string, policy *planstypes.Policy) (*projectstypes.MsgSetSubscriptionPolicyResponse, error) {
	msg := &projectstypes.MsgSetSubscriptionPolicy{
		Creator:  subkey,
		Policy:   policy,
		Projects: []string{projectID},
	}
	return ts.Servers.ProjectServer.SetSubscriptionPolicy(ts.GoCtx, msg)
}

// TxProjectSetPolicy: implement 'tx project set-policy'
func (ts *Tester) TxProjectSetPolicy(projectID, subkey string, policy *planstypes.Policy) (*projectstypes.MsgSetPolicyResponse, error) {
	msg := &projectstypes.MsgSetPolicy{
		Creator: subkey,
		Policy:  policy,
		Project: projectID,
	}
	return ts.Servers.ProjectServer.SetPolicy(ts.GoCtx, msg)
}

// TxPairingStakeProvider: implement 'tx pairing stake-provider'
func (ts *Tester) TxPairingStakeProvider(
	vault string,
	provider string,
	chainID string,
	amount sdk.Coin,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	description stakingtypes.Description,
) (*pairingtypes.MsgStakeProviderResponse, error) {
	val, _ := ts.GetAccount(VALIDATOR, 0)
	msg := &pairingtypes.MsgStakeProvider{
		Creator:            vault,
		Validator:          sdk.ValAddress(val.Addr).String(),
		ChainID:            chainID,
		Amount:             amount,
		Geolocation:        geoloc,
		Endpoints:          endpoints,
		DelegateLimit:      sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), sdk.ZeroInt()),
		DelegateCommission: 100,
		Address:            provider,
		Description:        description,
	}
	return ts.Servers.PairingServer.StakeProvider(ts.GoCtx, msg)
}

// TxPairingStakeProvider: implement 'tx pairing stake-provider'
func (ts *Tester) TxPairingStakeProviderFull(
	vault string,
	provider string,
	chainID string,
	amount sdk.Coin,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	commission uint64,
	delegateLimit uint64,
	moniker string,
	identity string,
	website string,
	securityContact string,
	descriptionDetails string,
) (*pairingtypes.MsgStakeProviderResponse, error) {
	val, _ := ts.GetAccount(VALIDATOR, 0)
	// if geoloc left zero, use default 1
	if geoloc == 0 {
		geoloc = 1
	}

	// if necessary, generate mock endpoints
	if endpoints == nil {
		apiInterfaces := []string{}
		for _, apiCollection := range ts.specs[chainID].ApiCollections {
			apiInterfaces = append(apiInterfaces, apiCollection.CollectionData.ApiInterface)
		}
		geolocations := planstypes.GetGeolocationsFromUint(geoloc)

		for _, geo := range geolocations {
			endpoint := epochstoragetypes.Endpoint{
				IPPORT:        "123",
				ApiInterfaces: apiInterfaces,
				Geolocation:   int32(geo),
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	description := stakingtypes.NewDescription(moniker, identity, website, securityContact, descriptionDetails)

	msg := &pairingtypes.MsgStakeProvider{
		Creator:            vault,
		Validator:          sdk.ValAddress(val.Addr).String(),
		ChainID:            chainID,
		Amount:             amount,
		Geolocation:        geoloc,
		Endpoints:          endpoints,
		DelegateLimit:      sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), sdk.NewIntFromUint64(delegateLimit)),
		DelegateCommission: commission,
		Address:            provider,
		Description:        description,
	}
	return ts.Servers.PairingServer.StakeProvider(ts.GoCtx, msg)
}

// TxPairingUnstakeProvider: implement 'tx pairing unstake-provider'
func (ts *Tester) TxPairingUnstakeProvider(
	vault string,
	chainID string,
) (*pairingtypes.MsgUnstakeProviderResponse, error) {
	val, _ := ts.GetAccount(VALIDATOR, 0)
	msg := &pairingtypes.MsgUnstakeProvider{
		Validator: sdk.ValAddress(val.Addr).String(),
		Creator:   vault,
		ChainID:   chainID,
	}
	return ts.Servers.PairingServer.UnstakeProvider(ts.GoCtx, msg)
}

// TxPairingRelayPayment: implement 'tx pairing relay-payment'
func (ts *Tester) TxPairingRelayPayment(provider string, rs ...*pairingtypes.RelaySession) (*pairingtypes.MsgRelayPaymentResponse, error) {
	msg := &pairingtypes.MsgRelayPayment{
		Creator:           provider,
		Relays:            rs,
		DescriptionString: "test",
	}
	return ts.Servers.PairingServer.RelayPayment(ts.GoCtx, msg)
}

// TxPairingFreezeProvider: implement 'tx pairing freeze'
func (ts *Tester) TxPairingFreezeProvider(addr, chainID string) (*pairingtypes.MsgFreezeProviderResponse, error) {
	msg := &pairingtypes.MsgFreezeProvider{
		Creator:  addr,
		ChainIds: lavaslices.Slice(chainID),
		Reason:   "test",
	}
	return ts.Servers.PairingServer.FreezeProvider(ts.GoCtx, msg)
}

// TxPairingUnfreezeProvider: implement 'tx pairing unfreeze'
func (ts *Tester) TxPairingUnfreezeProvider(addr, chainID string) (*pairingtypes.MsgUnfreezeProviderResponse, error) {
	msg := &pairingtypes.MsgUnfreezeProvider{
		Creator:  addr,
		ChainIds: lavaslices.Slice(chainID),
	}
	return ts.Servers.PairingServer.UnfreezeProvider(ts.GoCtx, msg)
}

func (ts *Tester) TxRewardsSetIprpcDataProposal(authority string, cost sdk.Coin, subs []string) (*rewardstypes.MsgSetIprpcDataResponse, error) {
	msg := rewardstypes.NewMsgSetIprpcData(authority, cost, subs)
	return ts.Servers.RewardsServer.SetIprpcData(ts.GoCtx, msg)
}

func (ts *Tester) TxRewardsFundIprpc(creator string, spec string, duration uint64, fund sdk.Coins) (*rewardstypes.MsgFundIprpcResponse, error) {
	msg := rewardstypes.NewMsgFundIprpc(creator, spec, duration, fund)
	return ts.Servers.RewardsServer.FundIprpc(ts.GoCtx, msg)
}

// TxCreateValidator: implement 'tx staking createvalidator' and bond its tokens
func (ts *Tester) TxCreateValidator(validator sigs.Account, amount math.Int) {
	consensusPowerTokens := ts.Keepers.StakingKeeper.TokensFromConsensusPower(ts.Ctx, 1)
	if amount.LT(consensusPowerTokens) {
		utils.LavaFormatWarning(`validator stake should usually be larger than the amount of tokens for one 
		unit of consensus power`,
			fmt.Errorf("validator stake might be too small"),
			utils.Attribute{Key: "consensus_power_tokens", Value: consensusPowerTokens.String()},
			utils.Attribute{Key: "validator_stake", Value: amount.String()},
		)
	}

	// create a validator
	msg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(validator.Addr),
		validator.PubKey,
		sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), amount),
		stakingtypes.Description{},
		stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1)),
		sdk.ZeroInt(),
	)
	require.Nil(ts.T, err)
	_, err = ts.Servers.StakingServer.CreateValidator(ts.GoCtx, msg)
	require.Nil(ts.T, err)
	ts.AdvanceBlock() // advance block to run staking keeper's endBlocker that makes the validator bonded
}

// TxDelegateValidator: implement 'tx staking delegate'
func (ts *Tester) TxDelegateValidator(delegator, validator sigs.Account, amount math.Int) (*stakingtypes.MsgDelegateResponse, error) {
	msg := stakingtypes.NewMsgDelegate(
		delegator.Addr,
		sdk.ValAddress(validator.Addr),
		sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), amount),
	)
	return ts.Servers.StakingServer.Delegate(ts.GoCtx, msg)
}

// TxReDelegateValidator: implement 'tx staking redelegate'
func (ts *Tester) TxReDelegateValidator(delegator, fromValidator, toValidator sigs.Account, amount math.Int) (*stakingtypes.MsgBeginRedelegateResponse, error) {
	msg := stakingtypes.NewMsgBeginRedelegate(
		delegator.Addr,
		sdk.ValAddress(fromValidator.Addr),
		sdk.ValAddress(toValidator.Addr),
		sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), amount),
	)
	rf := dualstakingante.NewRedelegationFlager(ts.Keepers.Dualstaking)
	err := rf.DisableRedelegationHooks(ts.Ctx, []sdk.Msg{msg})
	require.NoError(ts.T, err)
	return ts.Servers.StakingServer.BeginRedelegate(ts.GoCtx, msg)
}

// TxUnbondValidator: implement 'tx staking undond'
func (ts *Tester) TxUnbondValidator(delegator, validator sigs.Account, amount math.Int) (*stakingtypes.MsgUndelegateResponse, error) {
	msg := stakingtypes.NewMsgUndelegate(
		delegator.Addr,
		sdk.ValAddress(validator.Addr),
		sdk.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), amount),
	)
	return ts.Servers.StakingServer.Undelegate(ts.GoCtx, msg)
}

// TxUnbondValidator: implement 'tx staking undond'
func (ts *Tester) TxCancelUnbondValidator(delegator, validator sigs.Account, block int64, amount sdk.Coin) (*stakingtypes.MsgCancelUnbondingDelegationResponse, error) {
	msg := stakingtypes.NewMsgCancelUnbondingDelegation(
		delegator.Addr,
		sdk.ValAddress(validator.Addr),
		block,
		amount,
	)
	return ts.Servers.StakingServer.CancelUnbondingDelegation(ts.GoCtx, msg)
}

// QuerySubscriptionCurrent: implement 'q subscription current'
func (ts *Tester) QuerySubscriptionCurrent(subkey string) (*subscriptiontypes.QueryCurrentResponse, error) {
	msg := &subscriptiontypes.QueryCurrentRequest{
		Consumer: subkey,
	}
	return ts.Keepers.Subscription.Current(ts.GoCtx, msg)
}

// QuerySubscriptionListProjects: implement 'q subscription list-projects'
func (ts *Tester) QuerySubscriptionListProjects(subkey string) (*subscriptiontypes.QueryListProjectsResponse, error) {
	msg := &subscriptiontypes.QueryListProjectsRequest{
		Subscription: subkey,
	}
	return ts.Keepers.Subscription.ListProjects(ts.GoCtx, msg)
}

// QuerySubscriptionNextToMonthExpiry: implement 'q subscription next-to-month-expiry'
func (ts *Tester) QuerySubscriptionNextToMonthExpiry() (*subscriptiontypes.QueryNextToMonthExpiryResponse, error) {
	msg := &subscriptiontypes.QueryNextToMonthExpiryRequest{}
	return ts.Keepers.Subscription.NextToMonthExpiry(ts.GoCtx, msg)
}

// QueryProjectInfo implements 'q project info'
func (ts *Tester) QueryProjectInfo(projectID string) (*projectstypes.QueryInfoResponse, error) {
	msg := &projectstypes.QueryInfoRequest{Project: projectID}
	return ts.Keepers.Projects.Info(ts.GoCtx, msg)
}

// QueryProjectDeveloper implements 'q project developer'
func (ts *Tester) QueryProjectDeveloper(devkey string) (*projectstypes.QueryDeveloperResponse, error) {
	msg := &projectstypes.QueryDeveloperRequest{Developer: devkey}
	return ts.Keepers.Projects.Developer(ts.GoCtx, msg)
}

// QueryPairingGetPairing implements 'q pairing get-pairing'
func (ts *Tester) QueryPairingGetPairing(chainID, client string) (*pairingtypes.QueryGetPairingResponse, error) {
	msg := &pairingtypes.QueryGetPairingRequest{
		ChainID: chainID,
		Client:  client,
	}
	return ts.Keepers.Pairing.GetPairing(ts.GoCtx, msg)
}

// QueryPairingProviderPairingChance implements 'q pairing get-pairing'
func (ts *Tester) QueryPairingProviderPairingChance(provider string, chainID string, geolocation int32, cluster string) (*pairingtypes.QueryProviderPairingChanceResponse, error) {
	msg := &pairingtypes.QueryProviderPairingChanceRequest{
		Provider:    provider,
		ChainID:     chainID,
		Geolocation: geolocation,
		Cluster:     cluster,
	}
	return ts.Keepers.Pairing.ProviderPairingChance(ts.GoCtx, msg)
}

// QueryPairingProviders: implement 'q pairing providers'
func (ts *Tester) QueryPairingProviders(chainID string, frozen bool) (*pairingtypes.QueryProvidersResponse, error) {
	msg := &pairingtypes.QueryProvidersRequest{
		ChainID:    chainID,
		ShowFrozen: frozen,
	}
	return ts.Keepers.Pairing.Providers(ts.GoCtx, msg)
}

// QueryPairingProvider: implement 'q pairing provider'
func (ts *Tester) QueryPairingProvider(address string, chainID string) (*pairingtypes.QueryProviderResponse, error) {
	msg := &pairingtypes.QueryProviderRequest{
		Address: address,
		ChainID: chainID,
	}
	return ts.Keepers.Pairing.Provider(ts.GoCtx, msg)
}

// QueryPairingVerifyPairing implements 'q pairing verfy-pairing'
func (ts *Tester) QueryPairingVerifyPairing(chainID, client, provider string, block uint64) (*pairingtypes.QueryVerifyPairingResponse, error) {
	msg := &pairingtypes.QueryVerifyPairingRequest{
		ChainID:  chainID,
		Client:   client,
		Provider: provider,
		Block:    block,
	}
	return ts.Keepers.Pairing.VerifyPairing(ts.GoCtx, msg)
}

// QueryPairingEffectivePolicy implements 'q pairing effective-policy'
func (ts *Tester) QueryPairingEffectivePolicy(chainID, consumer string) (*pairingtypes.QueryEffectivePolicyResponse, error) {
	msg := &pairingtypes.QueryEffectivePolicyRequest{
		SpecID:   chainID,
		Consumer: consumer,
	}
	return ts.Keepers.Pairing.EffectivePolicy(ts.GoCtx, msg)
}

// QueryPairingProviderMonthlyPayout implements 'q pairing provider-monthly-payout'
func (ts *Tester) QueryPairingProviderMonthlyPayout(provider string) (*pairingtypes.QueryProviderMonthlyPayoutResponse, error) {
	msg := &pairingtypes.QueryProviderMonthlyPayoutRequest{
		Provider: provider,
	}
	return ts.Keepers.Pairing.ProviderMonthlyPayout(ts.GoCtx, msg)
}

// QueryPairingProviderEpochCu implements 'q pairing provider-epoch-cu'
func (ts *Tester) QueryPairingProviderEpochCu(provider string, project string, chainID string) (*pairingtypes.QueryProvidersEpochCuResponse, error) {
	msg := &pairingtypes.QueryProvidersEpochCuRequest{}
	return ts.Keepers.Pairing.ProvidersEpochCu(ts.GoCtx, msg)
}

// QueryPairingSubscriptionMonthlyPayout implements 'q pairing subscription-monthly-payout'
func (ts *Tester) QueryPairingSubscriptionMonthlyPayout(consumer string) (*pairingtypes.QuerySubscriptionMonthlyPayoutResponse, error) {
	msg := &pairingtypes.QuerySubscriptionMonthlyPayoutRequest{
		Consumer: consumer,
	}
	return ts.Keepers.Pairing.SubscriptionMonthlyPayout(ts.GoCtx, msg)
}

// QueryPairingVerifyPairing implements 'q dualstaking delegator-providers'
func (ts *Tester) QueryDualstakingDelegatorProviders(delegator string, withPending bool) (*dualstakingtypes.QueryDelegatorProvidersResponse, error) {
	msg := &dualstakingtypes.QueryDelegatorProvidersRequest{
		Delegator:   delegator,
		WithPending: withPending,
	}
	return ts.Keepers.Dualstaking.DelegatorProviders(ts.GoCtx, msg)
}

// QueryDualstakingProviderDelegators implements 'q dualstaking provider-delegators'
func (ts *Tester) QueryDualstakingProviderDelegators(provider string, withPending bool) (*dualstakingtypes.QueryProviderDelegatorsResponse, error) {
	msg := &dualstakingtypes.QueryProviderDelegatorsRequest{
		Provider:    provider,
		WithPending: withPending,
	}
	return ts.Keepers.Dualstaking.ProviderDelegators(ts.GoCtx, msg)
}

// QueryDualstakingDelegatorRewards implements 'q dualstaking delegator-rewards'
func (ts *Tester) QueryDualstakingDelegatorRewards(delegator string, provider string, chainID string) (*dualstakingtypes.QueryDelegatorRewardsResponse, error) {
	msg := &dualstakingtypes.QueryDelegatorRewardsRequest{
		Delegator: delegator,
		Provider:  provider,
		ChainId:   chainID,
	}
	return ts.Keepers.Dualstaking.DelegatorRewards(ts.GoCtx, msg)
}

// QueryFixationAllIndices implements 'q fixationstore all-indices'
func (ts *Tester) QueryFixationAllIndices(storeKey string, prefix string) (*fixationstoretypes.QueryAllIndicesResponse, error) {
	msg := &fixationstoretypes.QueryAllIndicesRequest{
		StoreKey: storeKey,
		Prefix:   prefix,
	}
	return ts.Keepers.FixationStoreKeeper.AllIndices(ts.GoCtx, msg)
}

// QueryStoreKeys implements 'q fixationstore store-keys'
func (ts *Tester) QueryStoreKeys() (*fixationstoretypes.QueryStoreKeysResponse, error) {
	msg := &fixationstoretypes.QueryStoreKeysRequest{}
	return ts.Keepers.FixationStoreKeeper.StoreKeys(ts.GoCtx, msg)
}

// QueryFixationVersions implements 'q fixationstore versions'
func (ts *Tester) QueryFixationVersions(storeKey string, prefix string, key string) (*fixationstoretypes.QueryVersionsResponse, error) {
	msg := &fixationstoretypes.QueryVersionsRequest{
		StoreKey: storeKey,
		Prefix:   prefix,
		Key:      key,
	}
	return ts.Keepers.FixationStoreKeeper.Versions(ts.GoCtx, msg)
}

// QueryFixationEntry implements 'q fixationstore entry'
func (ts *Tester) QueryFixationEntry(storeKey string, prefix string, key string, block uint64, hideData bool, stringData bool) (*fixationstoretypes.QueryEntryResponse, error) {
	msg := &fixationstoretypes.QueryEntryRequest{
		StoreKey:   storeKey,
		Prefix:     prefix,
		Key:        key,
		Block:      block,
		HideData:   hideData,
		StringData: stringData,
	}
	return ts.Keepers.FixationStoreKeeper.Entry(ts.GoCtx, msg)
}

// QueryRewardsPools implements 'q rewards pools'
func (ts *Tester) QueryRewardsPools() (*rewardstypes.QueryPoolsResponse, error) {
	msg := &rewardstypes.QueryPoolsRequest{}
	return ts.Keepers.Rewards.Pools(ts.GoCtx, msg)
}

// QueryRewardsBlockReward implements 'q rewards block-reward'
func (ts *Tester) QueryRewardsBlockReward() (*rewardstypes.QueryBlockRewardResponse, error) {
	msg := &rewardstypes.QueryBlockRewardRequest{}
	return ts.Keepers.Rewards.BlockReward(ts.GoCtx, msg)
}

// QueryRewardsShowIprpcData implements 'q rewards show-iprpc-data'
func (ts *Tester) QueryRewardsShowIprpcData() (*rewardstypes.QueryShowIprpcDataResponse, error) {
	msg := &rewardstypes.QueryShowIprpcDataRequest{}
	return ts.Keepers.Rewards.ShowIprpcData(ts.GoCtx, msg)
}

func (ts *Tester) QueryRewardsIprpcProviderRewardEstimation(provider string) (*rewardstypes.QueryIprpcProviderRewardEstimationResponse, error) {
	msg := &rewardstypes.QueryIprpcProviderRewardEstimationRequest{
		Provider: provider,
	}
	return ts.Keepers.Rewards.IprpcProviderRewardEstimation(ts.GoCtx, msg)
}

func (ts *Tester) QueryRewardsIprpcSpecReward(spec string) (*rewardstypes.QueryIprpcSpecRewardResponse, error) {
	msg := &rewardstypes.QueryIprpcSpecRewardRequest{
		Spec: spec,
	}
	return ts.Keepers.Rewards.IprpcSpecReward(ts.GoCtx, msg)
}

// block/epoch helpers

func (ts *Tester) BlockHeight() uint64 {
	return uint64(ts.Ctx.BlockHeight())
}

func (ts *Tester) BlockTime() time.Time {
	return ts.Ctx.BlockTime()
}

// blocks and epochs

func (ts *Tester) BlocksToSave() uint64 {
	blocksToSave, err := ts.Keepers.Epochstorage.BlocksToSave(ts.Ctx, ts.BlockHeight())
	if err != nil {
		panic("BlocksToSave: failed to fetch: " + err.Error())
	}
	return blocksToSave
}

func (ts *Tester) EpochsToSave(block ...uint64) uint64 {
	if len(block) == 0 {
		return ts.Keepers.Epochstorage.EpochsToSaveRaw(ts.Ctx)
	}
	epochsToSave, err := ts.Keepers.Epochstorage.EpochsToSave(ts.Ctx, ts.BlockHeight())
	if err != nil {
		panic("EpochsToSave: failed to fetch: " + err.Error())
	}
	return epochsToSave
}

func (ts *Tester) EpochBlocks(block ...uint64) uint64 {
	if len(block) == 0 {
		return ts.Keepers.Epochstorage.EpochBlocksRaw(ts.Ctx)
	}
	epoch, err := ts.Keepers.Epochstorage.EpochBlocks(ts.Ctx, block[0])
	if err != nil {
		panic("EpochBlocks: failed to fetch: " + err.Error())
	}
	return epoch
}

func (ts *Tester) EpochStart(block ...uint64) uint64 {
	if len(block) == 0 {
		return ts.Keepers.Epochstorage.GetEpochStart(ts.Ctx)
	}
	epoch, _, err := ts.Keepers.Epochstorage.GetEpochStartForBlock(ts.Ctx, block[0])
	if err != nil {
		panic("EpochStart: failed to fetch: " + err.Error())
	}
	return epoch
}

func (ts *Tester) GetNextEpoch() uint64 {
	epoch, err := ts.Keepers.Epochstorage.GetNextEpoch(ts.Ctx, ts.BlockHeight())
	if err != nil {
		panic("GetNextEpoch: failed to fetch: " + err.Error())
	}
	return epoch
}

func (ts *Tester) GetNextMonth(from time.Time) int64 {
	return utils.NextMonth(from).UTC().Unix()
}

func (ts *Tester) AdvanceToBlock(block uint64) {
	if block < ts.BlockHeight() {
		panic("AdvanceToBlock: block in the past: " +
			strconv.Itoa(int(block)) + "<" + strconv.Itoa(int(ts.BlockHeight())))
	}
	ts.AdvanceBlocks(block - ts.BlockHeight())
}

func (ts *Tester) AdvanceBlocks(count uint64, delta ...time.Duration) *Tester {
	for i := 0; i < int(count); i++ {
		ts.GoCtx = testkeeper.AdvanceBlock(ts.GoCtx, ts.Keepers, delta...)
	}
	ts.Ctx = sdk.UnwrapSDKContext(ts.GoCtx)
	return ts
}

func (ts *Tester) AdvanceBlock(delta ...time.Duration) *Tester {
	return ts.AdvanceBlocks(1, delta...)
}

func (ts *Tester) AdvanceEpochs(count uint64, delta ...time.Duration) *Tester {
	for i := 0; i < int(count); i++ {
		ts.GoCtx = testkeeper.AdvanceEpoch(ts.GoCtx, ts.Keepers, delta...)
	}
	ts.Ctx = sdk.UnwrapSDKContext(ts.GoCtx)
	return ts
}

func (ts *Tester) AdvanceEpoch(delta ...time.Duration) *Tester {
	return ts.AdvanceEpochs(1, delta...)
}

func (ts *Tester) AdvanceBlockUntilStale(delta ...time.Duration) *Tester {
	return ts.AdvanceBlocks(ts.BlocksToSave())
}

func (ts *Tester) AdvanceEpochUntilStale(delta ...time.Duration) *Tester {
	block := ts.BlockHeight() + ts.BlocksToSave()
	for block > ts.BlockHeight() {
		ts.AdvanceEpoch()
	}
	return ts
}

// AdvanceMonthFrom advanced blocks by given months, i.e. until block times
// exceeds at least that many months since the from argument (minus 5 seconds,
// so caller can control when to cross the desired time).
func (ts *Tester) AdvanceMonthsFrom(from time.Time, months int) *Tester {
	for next := from; months > 0; months -= 1 {
		next = next.AddDate(0, 1, 0)
		delta := next.Sub(ts.BlockTime())
		if months == 1 {
			delta -= 5 * time.Second
		}
		ts.AdvanceBlock(delta)
	}
	return ts
}

func (ts *Tester) BondDenom() string {
	return ts.Keepers.StakingKeeper.BondDenom(sdk.UnwrapSDKContext(ts.Ctx))
}

// AdvanceMonth advanced blocks by given months, like AdvanceMonthsFrom,
// starting from the current block's timestamp
func (ts *Tester) AdvanceMonths(months int) *Tester {
	return ts.AdvanceMonthsFrom(ts.BlockTime(), months)
}

func (ts *Tester) SetupForTests(getToTopMostPath string, specId string, validators int, subscriptions int, projectsInSubscription int, providers int) error {
	var balance int64 = 100000000000

	start := len(ts.Accounts(VALIDATOR))
	for i := 0; i < validators; i++ {
		acc, _ := ts.AddAccount(VALIDATOR, start+i, balance)
		ts.TxCreateValidator(acc, math.NewInt(balance))
	}

	sdkContext := sdk.UnwrapSDKContext(ts.Ctx)
	spec, err := testkeeper.GetASpec(specId, getToTopMostPath, &sdkContext, &ts.Keepers.Spec)
	if err != nil {
		return err
	}
	ts.AddSpec(spec.Index, spec)
	ts.Keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.Ctx), spec)
	start = len(ts.Accounts(CONSUMER))
	for i := 0; i < subscriptions; i++ {
		// setup consumer
		consumerAcc, consumerAddress := ts.AddAccount(CONSUMER, start+i, balance)
		ts.AddPlan("free", CreateMockPlan())
		ts.AddPolicy("mock", CreateMockPolicy())
		plan := ts.plans["free"]
		// subscribe consumer
		BuySubscription(ts.Ctx, *ts.Keepers, *ts.Servers, consumerAcc, plan.Index)
		// create projects:
		_, pd2both := ts.AddAccount(DEVELOPER, start+i, 10000)
		keys_1_admin_dev := []projectstypes.ProjectKey{
			projectstypes.NewProjectKey(pd2both).
				AddType(projectstypes.ProjectKey_ADMIN).
				AddType(projectstypes.ProjectKey_DEVELOPER),
		}
		policy := ts.Policy("mock")
		pd := projectstypes.ProjectData{
			Name:        "proj",
			Enabled:     true,
			ProjectKeys: keys_1_admin_dev,
			Policy:      &policy,
		}
		ts.AddProjectData("projdata", pd)
		err = ts.Keepers.Projects.CreateProject(ts.Ctx, consumerAddress, pd, plan)
		if err != nil {
			return err
		}
	}
	// setup providers
	start = len(ts.Accounts(PROVIDER))
	for i := 0; i < providers; i++ {
		acc, provider := ts.AddAccount(PROVIDER, start+i, balance)
		d := MockDescription()
		moniker := d.Moniker + strconv.Itoa(start+i)
		identity := d.Identity + strconv.Itoa(start+i)
		website := d.Website + strconv.Itoa(start+i)
		securityContact := d.SecurityContact + strconv.Itoa(start+i)
		details := d.Details + strconv.Itoa(start+i)
		err := ts.StakeProviderExtra(acc.GetVaultAddr(), provider, spec, spec.MinStakeProvider.Amount.Int64(), nil, 1, moniker, identity, website, securityContact, details)
		if err != nil {
			return err
		}
	}

	// advance for the staking to be valid
	ts.AdvanceEpoch()
	return nil
}

var sessionID uint64

func (ts *Tester) SendRelay(provider string, clientAcc sigs.Account, chainIDs []string, cuSum uint64) pairingtypes.MsgRelayPayment {
	var relays []*pairingtypes.RelaySession
	epoch := int64(ts.EpochStart(ts.BlockHeight()))

	// Create relay request. Change session ID each call to avoid double spending error
	for i, chainID := range chainIDs {
		relaySession := &pairingtypes.RelaySession{
			Provider:    provider,
			ContentHash: []byte("apiname"),
			SessionId:   sessionID,
			SpecId:      chainID,
			CuSum:       cuSum,
			Epoch:       epoch,
			RelayNum:    uint64(i),
		}
		sessionID += 1

		// Sign and send the payment requests
		sig, err := sigs.Sign(clientAcc.SK, *relaySession)
		relaySession.Sig = sig
		require.Nil(ts.T, err)

		relays = append(relays, relaySession)
	}

	return pairingtypes.MsgRelayPayment{Creator: provider, Relays: relays}
}

// DisableParticipationFees zeros validators and community participation fees
func (ts *Tester) DisableParticipationFees() {
	distParams := distributiontypes.DefaultParams()
	distParams.CommunityTax = sdk.ZeroDec()
	err := ts.Keepers.Distribution.SetParams(ts.Ctx, distParams)
	require.Nil(ts.T, err)
	require.True(ts.T, ts.Keepers.Distribution.GetParams(ts.Ctx).CommunityTax.IsZero())

	paramKey := string(rewardstypes.KeyValidatorsSubscriptionParticipation)
	zeroDec, err := sdk.ZeroDec().MarshalJSON()
	require.Nil(ts.T, err)
	paramVal := string(zeroDec)
	err = ts.TxProposalChangeParam(rewardstypes.ModuleName, paramKey, paramVal)
	require.Nil(ts.T, err)
	require.True(ts.T, ts.Keepers.Rewards.GetParams(ts.Ctx).ValidatorsSubscriptionParticipation.IsZero())
}
