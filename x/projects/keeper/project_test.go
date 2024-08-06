package keeper_test

import (
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/v2/testutil/common"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/utils/sigs"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
	"github.com/stretchr/testify/require"
)

const projectName = "mockname"

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}
	ts.AddPlan("free", common.CreateMockPlan())
	ts.AddPolicy("mock", common.CreateMockPolicy())
	return ts
}

func (ts *tester) setupProjectData() *tester {
	_, pd1adm := ts.AddAccount("pd_adm_", 1, 10000)
	_, pd2both := ts.AddAccount("pd_both_", 2, 10000)
	_, pd3adm := ts.AddAccount("pd_adm_", 3, 10000)
	_, pd3dev := ts.AddAccount("pd_dev_", 3, 10000)

	// admin key
	keys_1_admin := []types.ProjectKey{
		types.ProjectAdminKey(pd1adm),
	}
	// "both" key
	keys_1_admin_dev := []types.ProjectKey{
		types.NewProjectKey(pd2both).
			AddType(types.ProjectKey_ADMIN).
			AddType(types.ProjectKey_DEVELOPER),
	}
	// both (admin+developer) key
	keys_2_admin_and_dev := []types.ProjectKey{
		types.ProjectAdminKey(pd3adm),
		types.ProjectDeveloperKey(pd3dev),
	}

	policy := ts.Policy("mock")

	templates := []struct {
		code    string
		name    string
		enabled bool
		keys    []types.ProjectKey
		policy  *planstypes.Policy
	}{
		// project with admin key, enabled, has policy
		{"pd1", "mock_project_1", true, keys_1_admin, &policy},
		// project with "both" key, disabled, with policy
		{"pd2", "mock_project_2", false, keys_1_admin_dev, &policy},
		// project with 2 keys (one admin, one developer) disabled, no policy
		{"pd3", "mock_project_3", false, keys_2_admin_and_dev, nil},
	}

	for _, tt := range templates {
		pd := types.ProjectData{
			Name:        tt.name,
			Enabled:     tt.enabled,
			ProjectKeys: tt.keys,
			Policy:      tt.policy,
		}
		ts.AddProjectData(tt.code, pd)
	}

	return ts
}

func (ts *tester) isKeyInProject(index, key string, kind types.ProjectKey_Type) bool {
	resp, err := ts.QueryProjectInfo(index)
	require.Nil(ts.T, err, "project: "+index+", key: "+key)
	pk := resp.Project.GetKey(key)
	return pk.IsType(kind)
}

func TestCreateDefaultProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	plan := ts.Plan("free")
	_, sub1Addr := ts.Account("sub1")

	err := ts.Keepers.Projects.CreateAdminProject(ts.Ctx, sub1Addr, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	// subscription key is a developer in the default project
	res1, err := ts.QueryProjectDeveloper(sub1Addr)
	require.NoError(t, err)

	res2, err := ts.QueryProjectInfo(res1.Project.Index)
	require.NoError(t, err)

	require.Equal(t, res2.Project, res1.Project)
}

func TestCreateProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev
	ts.setupProjectData()

	plan := ts.Plan("free")
	projectData := ts.ProjectData("pd2")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_both_2")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	// test bad project names

	projectNameDefault := types.ADMIN_PROJECT_NAME
	projectNameLong := strings.Repeat(projectNameDefault, types.MAX_PROJECT_NAME_LEN+1)
	projectNameInvalid := "projectName,"

	nameTests := []struct {
		name        string
		projectName string
	}{
		{"bad projectName (duplicate)", projectData.Name},
		{"bad projectName (too long)", projectNameLong},
		{"bad projectName (contains comma)", projectNameInvalid},
		{"bad projectName (empty)", ""},
	}

	pd1 := projectData
	pd1.ProjectKeys = []types.ProjectKey{}

	for _, tt := range nameTests {
		t.Run(tt.name, func(t *testing.T) {
			pd1.Name = tt.projectName
			err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, pd1, plan)
			require.Error(t, err, tt.name)
		})
	}

	// test bad project keys

	pd2 := projectData
	pd2.Name = "nonDuplicateProjectName"
	pd2.ProjectKeys = []types.ProjectKey{
		types.ProjectDeveloperKey(sub1Addr),
		types.ProjectAdminKey(sub1Addr).AddType(0x4),
	}

	err = ts.TxSubscriptionAddProject(sub1Addr, pd2)
	require.Error(t, err)

	// subscription key is not a developer
	_, err = ts.QueryProjectDeveloper(sub1Addr)
	require.Error(t, err)

	res1, err := ts.QueryProjectDeveloper(adm1Addr)
	require.NoError(t, err)
	res2, err := ts.QueryProjectInfo(res1.Project.Index)
	require.NoError(t, err)

	require.Equal(t, res1.Project, res2.Project)

	_, err = ts.GetProjectForBlock(res1.Project.Index, ts.BlockHeight())
	require.NoError(t, err)

	// there should be one project key
	require.Equal(t, 1, len(res2.Project.ProjectKeys))

	// the project key is the admin key
	require.Equal(t, res2.Project.ProjectKeys[0].Key, adm1Addr)

	// the admin is both an admin and a developer
	require.True(t, res2.Project.ProjectKeys[0].IsType(types.ProjectKey_ADMIN))
	require.True(t, res2.Project.ProjectKeys[0].IsType(types.ProjectKey_DEVELOPER))
}

func TestProjectsServerAPI(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev

	_, sub1Addr := ts.Account("sub1")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	plan := ts.Plan("free")
	err := ts.TxProposalAddPlans(plan)
	require.NoError(t, err)

	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
	require.NoError(t, err)

	projectData := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)},
		Policy:      &plan.PlanPolicy,
	}

	err = ts.TxSubscriptionAddProject(sub1Addr, projectData)
	require.NoError(t, err)

	ts.AdvanceBlock()

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)
	err = ts.TxProjectAddKeys(projectID, sub1Addr, types.ProjectDeveloperKey(dev2Addr))
	require.NoError(t, err)

	ts.AdvanceBlock()

	require.True(t, ts.isKeyInProject(projectID, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID, dev2Addr, types.ProjectKey_DEVELOPER))

	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Project.ProjectKeys))
}

func TestDeleteProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev
	ts.setupProjectData()

	plan := ts.Plan("free")
	projectData := ts.ProjectData("pd2")

	_, sub1Addr := ts.Account("sub1")
	projectID := types.ProjectIndex(sub1Addr, projectData.Name)

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.NoError(t, err)

	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, "nonsense")
	require.Error(t, err)

	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectID)
	require.NoError(t, err)

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.NoError(t, err)

	ts.AdvanceEpoch()

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.Error(t, err)
}

func TestAddDelKeys(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev
	ts.setupProjectData()

	projectData := ts.ProjectData("pd3")
	plan := ts.Plan("free")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_3")
	_, dev1Addr := ts.Account("pd_dev_3")
	_, dev2Addr := ts.Account("dev1")
	_, dev3Addr := ts.Account("dev2")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	res, err := ts.QueryProjectDeveloper(dev1Addr)
	require.NoError(t, err)

	project := res.Project

	// add myself as admin (should fail)
	pk := types.ProjectAdminKey(dev1Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.Error(t, err)

	// admin key adds an invalid key (should fail)
	pk = types.NewProjectKey(dev2Addr).AddType(0x4)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.Error(t, err)

	// admin key adds a developer
	pk = types.ProjectDeveloperKey(dev2Addr)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.NoError(t, err)

	// developer tries to add an admin (should fail)
	pk = types.ProjectAdminKey(dev2Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.Error(t, err)

	// admin adding admin
	pk = types.ProjectAdminKey(dev1Addr)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.NoError(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(project.Index, dev2Addr, types.ProjectKey_DEVELOPER))

	ts.AdvanceBlock()

	// new admin adding another developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.NoError(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))

	// fetch project with new developer
	_, err = ts.QueryProjectDeveloper(dev3Addr)
	require.NoError(t, err)

	// developer delete admin (should fail)
	pk = types.ProjectAdminKey(adm1Addr)
	err = ts.TxProjectDelKeys(project.Index, dev2Addr, pk)
	require.Error(t, err)

	// developer delete developer (should fail)
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, dev2Addr, pk)
	require.Error(t, err)

	// new admin delete subscription owner admin (should fail)
	pk = types.ProjectAdminKey(sub1Addr)
	err = ts.TxProjectDelKeys(project.Index, adm1Addr, pk)
	require.Error(t, err)

	// subscription owner (admin) delete other admin
	pk = types.ProjectAdminKey(adm1Addr)
	err = ts.TxProjectDelKeys(project.Index, sub1Addr, pk)
	require.NoError(t, err)

	// admin delete developer after admin removed (should fail)
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, adm1Addr, pk)
	require.Error(t, err)

	// subscription owner (admin) delete developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, sub1Addr, pk)
	require.NoError(t, err)

	// deletion take effect in next epoch
	require.True(t, ts.isKeyInProject(project.Index, adm1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))
	ts.AdvanceEpoch()
	require.False(t, ts.isKeyInProject(project.Index, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))
}

func TestAddAdminInTwoProjects(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 admin, 0 devel
	ts.setupProjectData()

	projectData := ts.ProjectData("pd1")
	plan := ts.Plan("free")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_1")

	err := ts.Keepers.Projects.CreateAdminProject(ts.Ctx, sub1Addr, plan)
	require.NoError(t, err)

	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	_, err = ts.QueryProjectDeveloper(adm1Addr)
	require.Error(t, err)
	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.NoError(t, err)

	projectID := types.ProjectIndex(sub1Addr, types.ADMIN_PROJECT_NAME)
	require.Equal(t, res.Project.Index, projectID)
}

func TestSetPolicy(t *testing.T) {
	setPolicyTest(t, true)
}

func TestSetSubscriptionPolicy(t *testing.T) {
	setPolicyTest(t, false)
}

func setPolicyTest(t *testing.T, testAdminPolicy bool) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 1) // 1 sub, 0 admin, 1 data
	ts.setupProjectData()

	projectData := ts.ProjectData("pd1")
	plan := ts.Plan("free")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_1")
	_, dev1Addr := ts.Account("dev1")

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	pk := types.ProjectDeveloperKey(dev1Addr)
	err = ts.TxProjectAddKeys(projectID, adm1Addr, pk)
	require.NoError(t, err)

	spec := common.CreateMockSpec()
	ts.Keepers.Spec.SetSpec(ts.Ctx, spec)

	templates := []struct {
		name                         string
		creator                      string
		projectID                    string
		geolocation                  int32
		chainPolicies                []planstypes.ChainPolicy
		totalCuLimit                 uint64
		epochCuLimit                 uint64
		maxProvidersToPair           uint64
		setAdminPolicySuccess        bool
		setSubscriptionPolicySuccess bool
	}{
		{
			"valid policy (admin account)", adm1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, false,
		},
		{
			"valid policy (subscription account)", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, true,
		},
		{
			"bad creator (developer account -- not admin)", dev1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			"bad projectID (doesn't exist)", dev1Addr, "fakeProjectId", 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			"invalid geolocation (0)", dev1Addr, projectID, 0,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			// note: currently, we don't verify the chain policies
			"bad chainID (doesn't exist)", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: "LOL",
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"bad API (doesn't exist)", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{"lol"},
			}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"chainID and API not supported (exist in Lava's specs)", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: "ETH1",
				Apis:    []string{"eth_accounts"},
			}},
			100, 10, 3, true, true,
		},
		{
			"epoch CU larger than total CU", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			10, 100, 3, false, false,
		},
		{
			"bad maxProvidersToPair", sub1Addr, projectID, 1,
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 1, false, false,
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			newPolicy := planstypes.Policy{
				ChainPolicies:      tt.chainPolicies,
				GeolocationProfile: tt.geolocation,
				TotalCuLimit:       tt.totalCuLimit,
				EpochCuLimit:       tt.epochCuLimit,
				MaxProvidersToPair: tt.maxProvidersToPair,
			}

			if testAdminPolicy {
				_, err := ts.TxProjectSetPolicy(tt.projectID, tt.creator, &newPolicy)
				if tt.setAdminPolicySuccess {
					require.NoError(t, err)
					ts.AdvanceEpoch()
					proj, err := ts.GetProjectForBlock(tt.projectID, ts.BlockHeight())
					require.NoError(t, err)
					require.Equal(t, newPolicy, *proj.AdminPolicy)
				} else {
					require.Error(t, err)
				}
			} else {
				_, err := ts.TxProjectSetSubscriptionPolicy(tt.projectID, tt.creator, &newPolicy)
				if tt.setSubscriptionPolicySuccess {
					require.NoError(t, err)
					ts.AdvanceEpoch()
					proj, err := ts.GetProjectForBlock(tt.projectID, ts.BlockHeight())
					require.NoError(t, err)
					require.Equal(t, newPolicy, *proj.SubscriptionPolicy)
				} else {
					require.Error(t, err)
				}
			}
		})
	}
}

func TestChargeComputeUnits(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1) // 0 sub, 0 adm, 1 dev
	ts.setupProjectData()

	projectData := ts.ProjectData("pd1")
	plan := ts.Plan("free")

	_, sub1Addr := ts.Account("pd_adm_1")
	_, dev1Addr := ts.Account("dev1")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()
	block1 := ts.BlockHeight()

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)
	project, err := ts.GetProjectForBlock(projectID, block1)
	require.NoError(t, err)

	// first epoch to add some delay before adding the developer key.

	ts.AdvanceEpoch()
	block2 := ts.BlockHeight()

	// add developer key (created fixation)
	err = ts.TxProjectAddKeys(project.Index, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)

	// second epoch to move further, otherwise snapshot will affect the current new
	// dev key instead of creating a separate fixation version.

	ts.AdvanceEpoch()
	block3 := ts.BlockHeight()

	ts.Keepers.Projects.SnapshotSubscriptionProjects(ts.Ctx, sub1Addr, block3)

	// try to charge CUs: should update oldest and second-oldest entries, but not the latest
	// (because the latter is in a new snapshot)

	err = ts.Keepers.Projects.ChargeComputeUnitsToProject(ts.Ctx, project, block1, 1000)
	require.NoError(t, err)

	proj, err := ts.GetProjectForBlock(project.Index, block1)
	require.NoError(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block2)
	require.NoError(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block3)
	require.NoError(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)

	err = ts.Keepers.Projects.ChargeComputeUnitsToProject(ts.Ctx, project, block2, 1000)
	require.NoError(t, err)

	proj, err = ts.GetProjectForBlock(project.Index, block1)
	require.NoError(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block2)
	require.NoError(t, err)
	require.Equal(t, uint64(2000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block3)
	require.NoError(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)
}

func TestAddAfterDelKeys(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev

	_, sub1Addr := ts.Account("sub1")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	plan := ts.Plan("free")

	projectData1 := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{},
		Policy:      &plan.PlanPolicy,
	}
	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData1, plan)
	require.NoError(t, err)

	projectData2 := types.ProjectData{
		Name:        "mockname2",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData2, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	projectID1 := types.ProjectIndex(sub1Addr, projectData1.Name)
	projectID2 := types.ProjectIndex(sub1Addr, projectData2.Name)

	// add key
	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID2, dev2Addr, types.ProjectKey_DEVELOPER))

	// add same key to other project - should fail
	err = ts.TxProjectAddKeys(projectID2, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Error(t, err)

	// del key
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)

	// del takes effect in next epoch
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))
	ts.AdvanceEpoch()
	require.False(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))

	// add same key to other project
	err = ts.TxProjectAddKeys(projectID2, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)
	require.True(t, ts.isKeyInProject(projectID2, dev1Addr, types.ProjectKey_DEVELOPER))

	res, err := ts.QueryProjectDeveloper(dev1Addr)
	require.NoError(t, err)
	require.Equal(t, projectID2, res.Project.Index)
}

func TestAddDelKeysSameEpoch(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 2, 6) // 2 sub, 2 adm, 6 dev

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")
	_, adm1Addr := ts.Account("adm1")
	_, adm2Addr := ts.Account("adm2")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")
	_, dev3Addr := ts.Account("dev3")
	_, dev4Addr := ts.Account("dev4")
	_, dev5Addr := ts.Account("dev5")
	_, dev6Addr := ts.Account("dev6")

	plan := ts.Plan("free")

	projectData1 := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub1Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData1, plan)
	require.NoError(t, err)

	projectData2 := types.ProjectData{
		Name:        "mockname2",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub2Addr, projectData2, plan)
	require.NoError(t, err)

	ts.AdvanceBlock()

	projectID1 := types.ProjectIndex(sub1Addr, projectData1.Name)
	projectID2 := types.ProjectIndex(sub2Addr, projectData2.Name)

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev2Addr))
	require.NoError(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID1, dev2Addr, types.ProjectKey_DEVELOPER))

	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.NoError(t, err)
	require.Equal(t, 3, len(res.Project.ProjectKeys))

	// add twice - ok
	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.NoError(t, err)

	require.True(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// del twice - fail
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Error(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.Error(t, err)

	// del takes effect in next epoch
	ts.AdvanceEpoch()

	require.False(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// add, del, and add again in same epoch
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)

	err = ts.TxProjectAddKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.Error(t, err)

	ts.AdvanceEpoch()

	// add, del admin (admin should invalid immediately)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.Error(t, err)

	// del takes effect in next epoch
	ts.AdvanceEpoch()

	require.True(t, ts.isKeyInProject(projectID2, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID2, dev4Addr, types.ProjectKey_DEVELOPER))
	require.False(t, ts.isKeyInProject(projectID2, adm2Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID2, dev6Addr, types.ProjectKey_DEVELOPER))

	// add dev to two projects in same epoch (latter fails)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectDeveloperKey(dev5Addr))
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(projectID2, sub1Addr, types.ProjectDeveloperKey(dev5Addr))
	require.Error(t, err)
}

func TestDelKeysDelProjectSameEpoch(t *testing.T) {
	var err error

	ts := newTester(t)
	ts.SetupAccounts(1, 1, 2) // 1 sub, 1 adm, 2 dev

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("adm1")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	plan := ts.Plan("free")

	projectsData := []types.ProjectData{
		{
			Name:        "mockname1",
			Enabled:     true,
			ProjectKeys: []types.ProjectKey{types.ProjectAdminKey(adm1Addr)},
			Policy:      &plan.PlanPolicy,
		},
		{
			Name:        "mockname2",
			Enabled:     true,
			ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)},
			Policy:      &plan.PlanPolicy,
		},
		{
			Name:        "mockname3",
			Enabled:     true,
			ProjectKeys: []types.ProjectKey{types.ProjectAdminKey(adm1Addr)},
			Policy:      &plan.PlanPolicy,
		},
		{
			Name:        "mockname4",
			Enabled:     true,
			ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev2Addr)},
			Policy:      &plan.PlanPolicy,
		},
	}

	projectsID := make([]string, 4)

	for i := range projectsData {
		projectsID[i] = types.ProjectIndex(sub1Addr, projectsData[i].Name)
		err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectsData[i], plan)
		require.NoError(t, err)
	}

	ts.AdvanceEpoch()

	// part (1): delete keys then project

	// delete key from each project
	err = ts.TxProjectDelKeys(projectsID[0], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NoError(t, err)
	err = ts.TxProjectDelKeys(projectsID[1], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)

	// now delete the projects (double delete) in same epoch
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[0])
	require.NoError(t, err)
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[1])
	require.NoError(t, err)

	proj, err := ts.GetProjectForBlock(projectsID[0], ts.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.GetProjectForBlock(projectsID[1], ts.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch()

	proj, err = ts.GetProjectForBlock(projectsID[0], ts.BlockHeight())
	require.Error(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[1], ts.BlockHeight())
	require.Error(t, err)

	// should not panic
	ts.AdvanceBlocks(2 * ts.BlocksToSave())

	// part (2): delete project then keys

	// delete the projects
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[2])
	require.NoError(t, err)
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[3])
	require.NoError(t, err)

	// delete key from each project: should being  (project being deleted)
	err = ts.TxProjectDelKeys(projectsID[2], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Error(t, err)
	err = ts.TxProjectDelKeys(projectsID[3], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Error(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[2], ts.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.GetProjectForBlock(projectsID[3], ts.BlockHeight())
	require.NoError(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch()

	proj, err = ts.GetProjectForBlock(projectsID[2], ts.BlockHeight())
	require.Error(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[3], ts.BlockHeight())
	require.Error(t, err)

	// should not panic
	ts.AdvanceBlocks(2 * ts.BlocksToSave())
}

func TestAddDevKeyToDifferentProjectsInSameBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 0, 1) // 2 sub, 0 adm, 1 dev

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")
	_, dev1Addr := ts.Account("dev1")

	plan := ts.Plan("free")

	projectName1 := "mockname1"
	projectName2 := "mockname2"

	projectID1 := types.ProjectIndex(sub1Addr, projectName1)
	projectID2 := types.ProjectIndex(sub2Addr, projectName2)

	projectData1 := types.ProjectData{
		Name:        projectName1,
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub1Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData1, plan)
	require.NoError(t, err)

	projectData2 := types.ProjectData{
		Name:        projectName2,
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub2Addr, projectData2, plan)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)

	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Error(t, err) // developer was already added to the first project

	ts.AdvanceEpoch()

	res1, err := ts.QueryProjectDeveloper(sub1Addr)
	require.NoError(t, err)
	res2, err := ts.QueryProjectDeveloper(sub2Addr)
	require.NoError(t, err)

	require.Equal(t, 2, len(res1.Project.ProjectKeys))
	require.Equal(t, 1, len(res2.Project.ProjectKeys))
}

func TestSetPolicySelectedProviders(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	policy := ts.Policy("mock")

	allowed := planstypes.SELECTED_PROVIDERS_MODE_ALLOWED
	mixed := planstypes.SELECTED_PROVIDERS_MODE_MIXED
	exclusive := planstypes.SELECTED_PROVIDERS_MODE_EXCLUSIVE
	disabled := planstypes.SELECTED_PROVIDERS_MODE_DISABLED

	providersSets := []struct {
		planProviders []string
		subProviders  []string
		projProviders []string
	}{
		{[]string{}, []string{}, []string{}},
		{[]string{sub1Addr}, []string{sub1Addr}, []string{sub1Addr}},
		{[]string{sub1Addr}, []string{sub1Addr}, []string{}},
		{[]string{"lalala"}, []string{"lalala"}, []string{"lalala"}},
		{[]string{sub1Addr, sub1Addr}, []string{sub1Addr, sub1Addr}, []string{sub1Addr, sub1Addr}},
		{[]string{sub1Addr}, []string{}, []string{}},
	}

	templates := []struct {
		name            string
		planMode        planstypes.SELECTED_PROVIDERS_MODE
		subMode         planstypes.SELECTED_PROVIDERS_MODE
		projMode        planstypes.SELECTED_PROVIDERS_MODE
		providerSet     int
		planPolicyValid bool
		subPolicyValid  bool
		projPolicyValid bool
	}{
		{"ALLOWED mode happy flow", allowed, allowed, allowed, 0, true, true, true},
		{"ALLOWED mode non empty providers list", allowed, allowed, allowed, 1, false, false, false},

		{"EXCLUSIVE mode happy flow", exclusive, exclusive, exclusive, 2, true, true, true},
		{"EXCLUSIVE mode invalid providers addresses", exclusive, exclusive, exclusive, 3, false, false, false},
		{"EXCLUSIVE mode providers addresses duplicates", exclusive, exclusive, exclusive, 4, false, false, false},

		{"MIXED mode happy flow", mixed, mixed, mixed, 2, true, true, true},
		{"MIXED mode invalid providers addresses", mixed, mixed, mixed, 3, false, false, false},
		{"MIXED mode providers addresses duplicates", mixed, mixed, mixed, 4, false, false, false},

		{"DISABLED mode happy flow", disabled, mixed, mixed, 0, true, true, true},
		{"DISABLED mode non empty providers list", disabled, mixed, mixed, 5, false, true, true},
		{"DISABLED mode configured to proj/sub policy", mixed, disabled, disabled, 2, true, false, false},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			providersSet := providersSets[tt.providerSet]

			plan := ts.Plan("free")
			plan.PlanPolicy.SelectedProvidersMode = tt.planMode
			plan.PlanPolicy.SelectedProviders = providersSet.planProviders

			err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{plan}, false)
			if tt.planPolicyValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
			require.NoError(t, err)

			res, err := ts.QuerySubscriptionListProjects(sub1Addr)
			require.NoError(t, err)
			require.Equal(t, 1, len(res.Projects))

			admProject, err := ts.GetProjectForBlock(res.Projects[0], ts.BlockHeight())
			require.NoError(t, err)

			policy.SelectedProvidersMode = tt.projMode
			policy.SelectedProviders = providersSet.projProviders

			_, err = ts.TxProjectSetPolicy(admProject.Index, sub1Addr, &policy)
			if tt.projPolicyValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			policy.SelectedProvidersMode = tt.subMode
			policy.SelectedProviders = providersSet.subProviders

			_, err = ts.TxProjectSetSubscriptionPolicy(admProject.Index, sub1Addr, &policy)
			if tt.subPolicyValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestSetPolicyByGeolocation(t *testing.T) {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	// for convinience
	GLS := planstypes.Geolocation_value["GLS"]
	GL := planstypes.Geolocation_value["GL"]
	USE := planstypes.Geolocation_value["USE"]
	EU := planstypes.Geolocation_value["EU"]
	USE_EU := USE + EU

	// propose all plans
	freePlan := planstypes.Plan{
		Index:         "free",
		Block:         uint64(ctx.BlockHeight()),
		Price:         sdk.NewCoin(keepers.StakingKeeper.BondDenom(ctx), sdk.NewInt(1)),
		ProjectsLimit: 3,
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 4, // USE
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	basicPlan := planstypes.Plan{
		Index:         "basic",
		Block:         uint64(ctx.BlockHeight()),
		Price:         sdk.NewCoin(keepers.StakingKeeper.BondDenom(ctx), sdk.NewInt(1)),
		ProjectsLimit: 5,
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 0, // GLS
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	premiumPlan := planstypes.Plan{
		Index:         "premium",
		Block:         uint64(ctx.BlockHeight()),
		Price:         sdk.NewCoin(keepers.StakingKeeper.BondDenom(ctx), sdk.NewInt(1)),
		ProjectsLimit: 10,
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 65535, // GL
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	plans := []planstypes.Plan{freePlan, basicPlan, premiumPlan}
	err := testkeeper.SimulatePlansAddProposal(ctx, keepers.Plans, plans, false)
	require.NoError(t, err)

	freeUser := common.CreateNewAccount(_ctx, *keepers, 10000)
	basicUser := common.CreateNewAccount(_ctx, *keepers, 10000)
	premiumUser := common.CreateNewAccount(_ctx, *keepers, 10000)

	common.BuySubscription(_ctx, *keepers, *servers, freeUser, freePlan.Index)
	common.BuySubscription(_ctx, *keepers, *servers, basicUser, basicPlan.Index)
	common.BuySubscription(_ctx, *keepers, *servers, premiumUser, premiumPlan.Index)

	templates := []struct {
		name           string
		dev            sigs.Account
		planIndex      int
		setGeo         int32
		expectedGeo    int32
		setPolicyValid bool
		badGeo         bool // config that results in geo = 0
	}{
		// free plan users should not be able to change geo at all (default geo=USE)
		// note: setPolicy is valid, but the project's geo shouldn't change
		{"free plan invalid regular geo", freeUser, 0, EU, USE, true, true},
		{"free plan invalid geo GL", freeUser, 0, GL, USE, true, false},
		{"free plan invalid geo GLS", freeUser, 0, GLS, USE, false, true},

		// basic plan users should not be able to change geo at all (default geo=GLS)
		{"basic plan invalid regular geo", basicUser, 1, USE, GL, true, false},
		{"basic plan invalid geo GL", basicUser, 1, GL, GL, true, false},
		{"basic plan invalid geo GLS", basicUser, 1, GLS, GL, false, true},

		// premium/enterprise plan users should be able to change geo (default geo=GL)
		{"premium/enterprise plan - happy flow - regular geo", premiumUser, 2, USE, USE, true, false},
		{"premium/enterprise plan - happy flow - multiple regular geo", premiumUser, 2, USE_EU, USE_EU, true, false},
		{"premium/enterprise plan - happy flow - global geo", premiumUser, 2, GL, GL, true, false},
		{"premium/enterprise plan invalid geo GLS", premiumUser, 2, GLS, GL, false, true},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			devResponse, err := keepers.Projects.Developer(_ctx, &types.QueryDeveloperRequest{
				Developer: tt.dev.Addr.String(),
			})
			require.NoError(t, err)

			projIndex := devResponse.Project.Index

			_, err = servers.ProjectServer.SetPolicy(_ctx, &types.MsgSetPolicy{
				Creator: tt.dev.Addr.String(),
				Project: projIndex,
				Policy: &planstypes.Policy{
					GeolocationProfile: tt.setGeo,
					TotalCuLimit:       10,
					EpochCuLimit:       2,
					MaxProvidersToPair: 2,
				},
			})
			if tt.setPolicyValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				return
			}
			_ctx = testkeeper.AdvanceEpoch(_ctx, keepers) // apply the new policy

			devResponse, err = keepers.Projects.Developer(_ctx, &types.QueryDeveloperRequest{
				Developer: tt.dev.Addr.String(),
			})
			require.NoError(t, err)

			policies := []*planstypes.Policy{
				&plans[tt.planIndex].PlanPolicy,
				devResponse.Project.AdminPolicy,
				devResponse.Project.SubscriptionPolicy,
			}
			strictestGeo, err := keepers.Pairing.CalculateEffectiveGeolocationFromPolicies(policies)
			if !tt.badGeo {
				require.NoError(t, err)
				require.Equal(t, tt.expectedGeo, strictestGeo)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPendingProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0)

	_, sub := ts.Account("sub1")

	_, err := ts.TxSubscriptionBuy(sub, sub, "free", 1, false, false)
	require.NoError(t, err)

	res, err := ts.QuerySubscriptionListProjects(sub)
	require.NoError(t, err)
	projectID := res.Projects[0]

	adminPolicy := ts.Plan("free").PlanPolicy
	_, err = ts.TxProjectSetPolicy(projectID, sub, &adminPolicy)
	require.NoError(t, err)

	// we didn't advance an epoch yet so querying for the project should have a pending project
	infRes, err := ts.QueryProjectInfo(projectID)
	require.NoError(t, err)
	require.NotNil(t, infRes.PendingProject)
	pendingProjAdminPolicy := infRes.PendingProject.AdminPolicy
	require.True(t, adminPolicy.Equal(pendingProjAdminPolicy))

	devRes, err := ts.QueryProjectDeveloper(sub)
	require.NoError(t, err)
	require.NotNil(t, infRes.PendingProject)
	pendingProjAdminPolicy = devRes.PendingProject.AdminPolicy
	require.True(t, adminPolicy.Equal(pendingProjAdminPolicy))

	// advance an epoch to apply the new project settings, there should be no pending projects
	ts.AdvanceEpoch()
	infRes, err = ts.QueryProjectInfo(projectID)
	require.NoError(t, err)
	require.Nil(t, infRes.PendingProject)

	devRes, err = ts.QueryProjectDeveloper(sub)
	require.NoError(t, err)
	require.Nil(t, devRes.PendingProject)
}

// TestMaxKeysInProject tests that the max amount of keys in project is enforced as expected
// scenarios:
// 1. add keys to existing project and try to exceed max amount
// 2. delete one key (from project with max keys) and make sure you can add one more key
// 3. try to create a project with more keys than max to begin with
func TestMaxKeysInProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, types.MAX_KEYS_AMOUNT+1, 0)

	// create dummy keys
	var dummyKeys []types.ProjectKey
	for i := 1; i <= types.MAX_KEYS_AMOUNT+1; i++ {
		_, admin := ts.Account("adm" + strconv.Itoa(i))
		dummyKeys = append(dummyKeys, types.ProjectAdminKey(admin))
	}

	// buy subscription. has one key (auto-generated admin key)
	_, sub := ts.Account("sub1")
	_, err := ts.TxSubscriptionBuy(sub, sub, "free", 1, false, false)
	require.NoError(t, err)
	res, err := ts.QueryProjectDeveloper(sub)
	require.NoError(t, err)
	proj := res.Project

	// try adding more keys than allowed at once (should fail - one key too much)
	err = ts.TxProjectAddKeys(proj.Index, sub, dummyKeys...)
	require.Error(t, err)

	// add MAX_KEYS_AMOUNT-1, should succeed
	err = ts.TxProjectAddKeys(proj.Index, sub, dummyKeys[2:]...)
	require.NoError(t, err)

	// try to delete more keys than allowed, should fail
	err = ts.TxProjectDelKeys(proj.Index, sub, dummyKeys...)
	require.Error(t, err)

	// delete key and immediately try to add key - should fail since deletion is applied on next epoch
	err = ts.TxProjectDelKeys(proj.Index, sub, dummyKeys[2])
	require.NoError(t, err)
	err = ts.TxProjectAddKeys(proj.Index, sub, dummyKeys[0])
	require.Error(t, err)

	// wait an epoch and add the previosly added key, should succeed
	ts.AdvanceEpoch()
	err = ts.TxProjectAddKeys(proj.Index, sub, dummyKeys[2])
	require.NoError(t, err)

	// try to add a new project with more keys than allowed, should fail
	err = ts.TxSubscriptionAddProject(sub, types.ProjectData{
		Name:        "dummy",
		ProjectKeys: dummyKeys,
	})
	require.Error(t, err)
}
