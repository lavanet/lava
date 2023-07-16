package common

import (
	"context"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptionkeeper "github.com/lavanet/lava/x/subscription/keeper"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
)

type Tester struct {
	T *testing.T

	GoCtx   context.Context
	Ctx     sdk.Context
	Servers *testkeeper.Servers
	Keepers *testkeeper.Keepers

	accounts map[string]Account
	plans    map[string]planstypes.Plan
	policies map[string]planstypes.Policy
	projects map[string]projectstypes.ProjectData
	specs    map[string]spectypes.Spec
}

func NewTester(t *testing.T) *Tester {
	servers, keepers, GoCtx := testkeeper.InitAllKeepers(t)

	ts := &Tester{
		T:       t,
		GoCtx:   GoCtx,
		Ctx:     sdk.UnwrapSDKContext(GoCtx),
		Servers: servers,
		Keepers: keepers,

		accounts: make(map[string]Account),
		plans:    make(map[string]planstypes.Plan),
		policies: make(map[string]planstypes.Policy),
		projects: make(map[string]projectstypes.ProjectData),
		specs:    make(map[string]spectypes.Spec),
	}

	// AdvanceBlock() and AdvanceEpoch() always use the current time for the
	// first block (and ignores the time delta arg if given); So call it here
	// to generate a first timestamp and avoid any subsequent call with detla
	// argument call having the delta ignored.
	ts.AdvanceEpoch()

	return ts
}

func (ts *Tester) AddAccount(name string, balance int64) *Tester {
	ts.accounts[name] = CreateNewAccount(ts.GoCtx, *ts.Keepers, balance)
	return ts
}

func (ts *Tester) SetupAccounts(numSub, numAdm, numDev int) *Tester {
	for i := 0; i < numSub; i++ {
		name := "sub" + strconv.Itoa(i+1)
		ts.AddAccount(name, 20000)
	}
	for i := 0; i < numAdm; i++ {
		name := "adm" + strconv.Itoa(i+1)
		ts.AddAccount(name, 10000)
	}
	for i := 0; i < numDev; i++ {
		name := "dev" + strconv.Itoa(i+1)
		ts.AddAccount(name, 10000)
	}

	return ts
}

func (ts *Tester) Account(name string) (Account, string) {
	account, ok := ts.accounts[name]
	if !ok {
		panic("tester: unknown account: '" + name + "'")
	}
	return account, account.Addr.String()
}

func (ts *Tester) AddPlan(name string, plan planstypes.Plan) *Tester {
	err := ts.Keepers.Plans.AddPlan(ts.Ctx, plan)
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

// keeper helpers

func (ts *Tester) FindPlan(index string, block uint64) (planstypes.Plan, bool) {
	return ts.Keepers.Plans.FindPlan(ts.Ctx, index, block)
}

func (ts *Tester) GetProjectForBlock(projectID string, block uint64) (projectstypes.Project, error) {
	return ts.Keepers.Projects.GetProjectForBlock(ts.Ctx, projectID, block)
}

func (ts *Tester) GetProjectForDeveloper(devkey string, optional ...uint64) (projectstypes.Project, error) {
	block := ts.BlockHeight()
	if len(optional) > 1 {
		panic("GetProjectForDeveloper: more than one optional arg")
	}
	if len(optional) > 0 {
		block = optional[0]
	}
	return ts.Keepers.Projects.GetProjectForDeveloper(ts.Ctx, devkey, block)
}

func (ts *Tester) GetProjectDeveloperData(devkey string, block uint64) (projectstypes.ProtoDeveloperData, error) {
	return ts.Keepers.Projects.GetProjectDeveloperData(ts.Ctx, devkey, block)
}

// proposals, transactions, queries

func (ts *Tester) TxProposalAddPlans(plans ...planstypes.Plan) error {
	return testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans)
}

func (ts *Tester) TxProposalDelPlans(indices ...string) error {
	return testkeeper.SimulatePlansDelProposal(ts.Ctx, ts.Keepers.Plans, indices)
}

func (ts *Tester) TxProposalAddSpecs(specs ...spectypes.Spec) error {
	return testkeeper.SimulateSpecAddProposal(ts.Ctx, ts.Keepers.Spec, specs)
}

// TxSubscriptionBuy: implement 'tx subscription buy'
func (ts *Tester) TxSubscriptionBuy(creator, consumer string, plan string, months int) (*subscriptiontypes.MsgBuyResponse, error) {
	msg := &subscriptiontypes.MsgBuy{
		Creator:  creator,
		Consumer: consumer,
		Index:    plan,
		Duration: uint64(months),
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

// TxSubscriptionAddProject: implement 'tx subscription del-project'
func (ts *Tester) TxSubscriptionDelProject(creator string, projectID string) error {
	msg := &subscriptiontypes.MsgDelProject{
		Creator: creator,
		Name:    projectID,
	}
	_, err := ts.Servers.SubscriptionServer.DelProject(ts.GoCtx, msg)
	return err
}

func (ts *Tester) TxProjectSetSubscriptionPolicy(projectID string, subkey string, policy planstypes.Policy) (*projectstypes.MsgSetSubscriptionPolicyResponse, error) {
	msg := projectstypes.MsgSetSubscriptionPolicy{
		Creator:  subkey,
		Policy:   policy,
		Projects: []string{projectID},
	}
	return ts.Servers.ProjectServer.SetSubscriptionPolicy(ts.GoCtx, &msg)
}

func (ts *Tester) TxProjectSetProjectPolicy(projectID string, subkey string, policy planstypes.Policy) (*projectstypes.MsgSetPolicyResponse, error) {
	msg := projectstypes.MsgSetPolicy{
		Creator: subkey,
		Policy:  policy,
		Project: projectID,
	}
	return ts.Servers.ProjectServer.SetPolicy(ts.GoCtx, &msg)
}

// QuerySubscriptionListProjects: implement 'q subscription list-projects'
func (ts *Tester) QuerySubscriptionListProjects(subkey string) (*subscriptiontypes.QueryListProjectsResponse, error) {
	msg := &subscriptiontypes.QueryListProjectsRequest{
		Subscription: subkey,
	}
	return ts.Keepers.Subscription.ListProjects(ts.GoCtx, msg)
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
	return ts.AdvanceBlocks(types.STALE_ENTRY_TIME)
}

func (ts *Tester) AdvanceEpochUntilStale(delta ...time.Duration) *Tester {
	block := ts.BlockHeight() + types.STALE_ENTRY_TIME
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
