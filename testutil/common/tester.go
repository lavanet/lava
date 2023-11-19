package common

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptionkeeper "github.com/lavanet/lava/x/subscription/keeper"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
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
	PROVIDER  string = "provider"
	CONSUMER  string = "consumer"
	VALIDATOR string = "validator"
)

func NewTester(t *testing.T) *Tester {
	ts := NewTesterRaw(t)

	// AdvanceBlock() and AdvanceEpoch() always use the current time for the
	// first block (and ignores the time delta arg if given); So call it here
	// to generate a first timestamp and avoid any subsequent call with detla
	// argument call having the delta ignored.
	ts.AdvanceEpoch()

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
	ts.accounts[name] = CreateNewAccount(ts.GoCtx, *ts.Keepers, balance)
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

func (ts *Tester) StakeProvider(addr string, spec spectypes.Spec, amount int64) error {
	return ts.StakeProviderExtra(addr, spec, amount, nil, 0, "prov")
}

func (ts *Tester) StakeProviderExtra(
	addr string,
	spec spectypes.Spec,
	amount int64,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
) error {
	// if geoloc left zero, use default 1
	if geoloc == 0 {
		geoloc = 1
	}

	// if necessary, generate mock endpoints
	if endpoints == nil {
		apiInterface := spec.ApiCollections[0].CollectionData.ApiInterface
		geolocations := planstypes.GetGeolocationsFromUint(geoloc)

		for _, geo := range geolocations {
			endpoint := epochstoragetypes.Endpoint{
				IPPORT:        "123",
				ApiInterfaces: []string{apiInterface},
				Geolocation:   int32(geo),
			}
			endpoints = append(endpoints, endpoint)
		}
	}

	stake := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))
	_, err := ts.TxPairingStakeProvider(addr, spec.Name, stake, endpoints, geoloc, moniker)

	return err
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
	return epochstoragetypes.TokenDenom
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

func NewCoin(amount int64) sdk.Coin {
	return sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))
}

func NewCoins(amount ...int64) []sdk.Coin {
	return slices.Map(amount, NewCoin)
}

// keeper helpers

func (ts *Tester) GetBalance(accAddr sdk.AccAddress) int64 {
	denom := epochstoragetypes.TokenDenom
	return ts.Keepers.BankKeeper.GetBalance(ts.Ctx, accAddr, denom).Amount.Int64()
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
	msg := &dualstakingtypes.MsgDelegate{
		Creator:  creator,
		Provider: provider,
		ChainID:  chainID,
		Amount:   amount,
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
	msg := &dualstakingtypes.MsgUnbond{
		Creator:  creator,
		Provider: provider,
		ChainID:  chainID,
		Amount:   amount,
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
func (ts *Tester) TxSubscriptionBuy(creator, consumer, plan string, months int, autoRenewal bool) (*subscriptiontypes.MsgBuyResponse, error) {
	msg := &subscriptiontypes.MsgBuy{
		Creator:     creator,
		Consumer:    consumer,
		Index:       plan,
		Duration:    uint64(months),
		AutoRenewal: autoRenewal,
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
func (ts *Tester) TxSubscriptionDelProject(creator, projectID string) error {
	msg := &subscriptiontypes.MsgDelProject{
		Creator: creator,
		Name:    projectID,
	}
	_, err := ts.Servers.SubscriptionServer.DelProject(ts.GoCtx, msg)
	return err
}

// TxSubscriptionAutoRenewal: implement 'tx subscription auto-renewal'
func (ts *Tester) TxSubscriptionAutoRenewal(creator string, enable bool) error {
	msg := &subscriptiontypes.MsgAutoRenewal{
		Creator: creator,
		Enable:  enable,
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
func (ts *Tester) TxProjectSetSubscriptionPolicy(projectID, subkey string, policy planstypes.Policy) (*projectstypes.MsgSetSubscriptionPolicyResponse, error) {
	msg := &projectstypes.MsgSetSubscriptionPolicy{
		Creator:  subkey,
		Policy:   policy,
		Projects: []string{projectID},
	}
	return ts.Servers.ProjectServer.SetSubscriptionPolicy(ts.GoCtx, msg)
}

// TxProjectSetPolicy: implement 'tx project set-policy'
func (ts *Tester) TxProjectSetPolicy(projectID, subkey string, policy planstypes.Policy) (*projectstypes.MsgSetPolicyResponse, error) {
	msg := &projectstypes.MsgSetPolicy{
		Creator: subkey,
		Policy:  policy,
		Project: projectID,
	}
	return ts.Servers.ProjectServer.SetPolicy(ts.GoCtx, msg)
}

// TxPairingStakeProvider: implement 'tx pairing stake-provider'
func (ts *Tester) TxPairingStakeProvider(
	addr string,
	chainID string,
	amount sdk.Coin,
	endpoints []epochstoragetypes.Endpoint,
	geoloc int32,
	moniker string,
) (*pairingtypes.MsgStakeProviderResponse, error) {
	msg := &pairingtypes.MsgStakeProvider{
		Creator:            addr,
		ChainID:            chainID,
		Amount:             amount,
		Geolocation:        geoloc,
		Endpoints:          endpoints,
		Moniker:            moniker,
		DelegateLimit:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()),
		DelegateCommission: 100,
	}
	return ts.Servers.PairingServer.StakeProvider(ts.GoCtx, msg)
}

// TxPairingUnstakeProvider: implement 'tx pairing unstake-provider'
func (ts *Tester) TxPairingUnstakeProvider(
	addr string,
	chainID string,
) (*pairingtypes.MsgUnstakeProviderResponse, error) {
	msg := &pairingtypes.MsgUnstakeProvider{
		Creator: addr,
		ChainID: chainID,
	}
	return ts.Servers.PairingServer.UnstakeProvider(ts.GoCtx, msg)
}

// TxPairingRelayPayment: implement 'tx pairing relay-payment'
func (ts *Tester) TxPairingRelayPayment(addr string, rs ...*pairingtypes.RelaySession) (*pairingtypes.MsgRelayPaymentResponse, error) {
	msg := &pairingtypes.MsgRelayPayment{
		Creator: addr,
		Relays:  rs,
	}
	return ts.Servers.PairingServer.RelayPayment(ts.GoCtx, msg)
}

// TxPairingFreezeProvider: implement 'tx pairing freeze'
func (ts *Tester) TxPairingFreezeProvider(addr, chainID string) (*pairingtypes.MsgFreezeProviderResponse, error) {
	msg := &pairingtypes.MsgFreezeProvider{
		Creator:  addr,
		ChainIds: slices.Slice(chainID),
	}
	return ts.Servers.PairingServer.FreezeProvider(ts.GoCtx, msg)
}

// TxPairingUnfreezeProvider: implement 'tx pairing unfreeze'
func (ts *Tester) TxPairingUnfreezeProvider(addr, chainID string) (*pairingtypes.MsgUnfreezeProviderResponse, error) {
	msg := &pairingtypes.MsgUnfreezeProvider{
		Creator:  addr,
		ChainIds: slices.Slice(chainID),
	}
	return ts.Servers.PairingServer.UnfreezeProvider(ts.GoCtx, msg)
}

// TxCreateValidator: implement 'tx staking createvalidator'
func (ts *Tester) TxCreateValidator(validator sigs.Account, amount sdk.Int) (*stakingtypes.MsgCreateValidatorResponse, error) {
	msg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(validator.Addr),
		validator.PubKey,
		sdk.NewCoin(epochstoragetypes.TokenDenom, amount),
		stakingtypes.Description{},
		stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1), sdk.NewDecWithPrec(1, 1)),
		sdk.ZeroInt(),
	)
	require.Nil(ts.T, err)
	return ts.Servers.StakingServer.CreateValidator(ts.GoCtx, msg)
}

// TxDelegateValidator: implement 'tx staking delegate'
func (ts *Tester) TxDelegateValidator(delegator, validator sigs.Account, amount sdk.Int) (*stakingtypes.MsgDelegateResponse, error) {
	msg := stakingtypes.NewMsgDelegate(
		delegator.Addr,
		sdk.ValAddress(validator.Addr),
		sdk.NewCoin(epochstoragetypes.TokenDenom, amount),
	)
	return ts.Servers.StakingServer.Delegate(ts.GoCtx, msg)
}

// TxReDelegateValidator: implement 'tx staking redelegate'
func (ts *Tester) TxReDelegateValidator(delegator, fromValidator, toValidator sigs.Account, amount sdk.Int) (*stakingtypes.MsgBeginRedelegateResponse, error) {
	msg := stakingtypes.NewMsgBeginRedelegate(
		delegator.Addr,
		sdk.ValAddress(fromValidator.Addr),
		sdk.ValAddress(toValidator.Addr),
		sdk.NewCoin(epochstoragetypes.TokenDenom, amount),
	)
	return ts.Servers.StakingServer.BeginRedelegate(ts.GoCtx, msg)
}

// TxUnbondValidator: implement 'tx staking undond'
func (ts *Tester) TxUnbondValidator(delegator, validator sigs.Account, amount sdk.Int) (*stakingtypes.MsgUndelegateResponse, error) {
	msg := stakingtypes.NewMsgUndelegate(
		delegator.Addr,
		sdk.ValAddress(validator.Addr),
		sdk.NewCoin(epochstoragetypes.TokenDenom, amount),
	)
	return ts.Servers.StakingServer.Undelegate(ts.GoCtx, msg)
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

// QueryPairingListEpochPayments implements 'q pairing list-epoch-payments'
func (ts *Tester) QueryPairingListEpochPayments() (*pairingtypes.QueryAllEpochPaymentsResponse, error) {
	msg := &pairingtypes.QueryAllEpochPaymentsRequest{}
	return ts.Keepers.Pairing.EpochPaymentsAll(ts.GoCtx, msg)
}

// QueryPairingProviders: implement 'q pairing providers'
func (ts *Tester) QueryPairingProviders(chainID string, frozen bool) (*pairingtypes.QueryProvidersResponse, error) {
	msg := &pairingtypes.QueryProvidersRequest{
		ChainID:    chainID,
		ShowFrozen: frozen,
	}
	return ts.Keepers.Pairing.Providers(ts.GoCtx, msg)
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
		ts.GoCtx = testkeeper.AdvanceEpoch(ts.GoCtx, ts.Keepers)
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
		next = subscriptionkeeper.NextMonth(next)
		fmt.Printf("next: %v\n", next.Unix())
		delta := next.Sub(ts.BlockTime())
		if months == 1 {
			delta -= 5 * time.Second
		}
		ts.AdvanceBlock(delta)
	}
	return ts
}

// AdvanceMonth advanced blocks by given months, like AdvanceMonthsFrom,
// starting from the current block's timestamp
func (ts *Tester) AdvanceMonths(months int) *Tester {
	return ts.AdvanceMonthsFrom(ts.BlockTime(), months)
}
