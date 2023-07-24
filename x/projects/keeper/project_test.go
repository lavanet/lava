package keeper_test

import (
	"strings"
	"testing"

	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

const projectName = "mockname"

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}
	ts.AddPlan("mock", common.CreateMockPlan())
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

	plan := ts.Plan("mock")
	_, sub1Addr := ts.Account("sub1")

	err := ts.Keepers.Projects.CreateAdminProject(ts.Ctx, sub1Addr, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	// subscription key is a developer in the default project
	res1, err := ts.QueryProjectDeveloper(sub1Addr)
	require.Nil(t, err)

	res2, err := ts.QueryProjectInfo(res1.Project.Index)
	require.Nil(t, err)

	require.Equal(t, res2.Project, res1.Project)
}

func TestCreateProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev
	ts.setupProjectData()

	plan := ts.Plan("mock")
	projectData := ts.ProjectData("pd2")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_both_2")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

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
			require.NotNil(t, err, tt.name)
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
	require.NotNil(t, err)

	// subscription key is not a developer
	_, err = ts.QueryProjectDeveloper(sub1Addr)
	require.NotNil(t, err)

	res1, err := ts.QueryProjectDeveloper(adm1Addr)
	require.Nil(t, err)
	res2, err := ts.QueryProjectInfo(res1.Project.Index)
	require.Nil(t, err)

	require.Equal(t, res1.Project, res2.Project)

	_, err = ts.GetProjectForBlock(res1.Project.Index, ts.BlockHeight())
	require.Nil(t, err)

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

	plan := ts.Plan("mock")
	err := ts.TxProposalAddPlans(plan)
	require.Nil(t, err)

	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)

	projectData := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)},
		Policy:      &plan.PlanPolicy,
	}

	err = ts.TxSubscriptionAddProject(sub1Addr, projectData)
	require.Nil(t, err)

	ts.AdvanceBlock()

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)
	err = ts.TxProjectAddKeys(projectID, sub1Addr, types.ProjectDeveloperKey(dev2Addr))
	require.Nil(t, err)

	ts.AdvanceBlock()

	require.True(t, ts.isKeyInProject(projectID, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID, dev2Addr, types.ProjectKey_DEVELOPER))

	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Project.ProjectKeys))
}

func TestDeleteProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev
	ts.setupProjectData()

	plan := ts.Plan("mock")
	projectData := ts.ProjectData("pd2")

	_, sub1Addr := ts.Account("sub1")
	projectID := types.ProjectIndex(sub1Addr, projectData.Name)

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.Nil(t, err)

	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, "nonsense")
	require.NotNil(t, err)

	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectID)
	require.Nil(t, err)

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.Nil(t, err)

	ts.AdvanceEpoch()

	_, err = ts.GetProjectForBlock(projectID, ts.BlockHeight())
	require.NotNil(t, err)
}

func TestAddDelKeys(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 2) // 1 sub, 0 adm, 2 dev
	ts.setupProjectData()

	projectData := ts.ProjectData("pd3")
	plan := ts.Plan("mock")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_3")
	_, dev1Addr := ts.Account("pd_dev_3")
	_, dev2Addr := ts.Account("dev1")
	_, dev3Addr := ts.Account("dev2")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	res, err := ts.QueryProjectDeveloper(dev1Addr)
	require.Nil(t, err)

	project := res.Project

	// add myself as admin (should fail)
	pk := types.ProjectAdminKey(dev1Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.NotNil(t, err)

	// admin key adds an invalid key (should fail)
	pk = types.NewProjectKey(dev2Addr).AddType(0x4)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.NotNil(t, err)

	// admin key adds a developer
	pk = types.ProjectDeveloperKey(dev2Addr)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.Nil(t, err)

	// developer tries to add an admin (should fail)
	pk = types.ProjectAdminKey(dev2Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.NotNil(t, err)

	// admin adding admin
	pk = types.ProjectAdminKey(dev1Addr)
	err = ts.TxProjectAddKeys(project.Index, adm1Addr, pk)
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(project.Index, dev2Addr, types.ProjectKey_DEVELOPER))

	ts.AdvanceBlock()

	// new admin adding another developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectAddKeys(project.Index, dev1Addr, pk)
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))

	// fetch project with new developer
	_, err = ts.QueryProjectDeveloper(dev3Addr)
	require.Nil(t, err)

	// developer delete admin (should fail)
	pk = types.ProjectAdminKey(adm1Addr)
	err = ts.TxProjectDelKeys(project.Index, dev2Addr, pk)
	require.NotNil(t, err)

	// developer delete developer (should fail)
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, dev2Addr, pk)
	require.NotNil(t, err)

	// new admin delete subscription owner admin (should fail)
	pk = types.ProjectAdminKey(sub1Addr)
	err = ts.TxProjectDelKeys(project.Index, adm1Addr, pk)
	require.NotNil(t, err)

	// subscription owner (admin) delete other admin
	pk = types.ProjectAdminKey(adm1Addr)
	err = ts.TxProjectDelKeys(project.Index, sub1Addr, pk)
	require.Nil(t, err)

	// admin delete developer after admin removed (should fail)
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, adm1Addr, pk)
	require.NotNil(t, err)

	// subscription owner (admin) delete developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.TxProjectDelKeys(project.Index, sub1Addr, pk)
	require.Nil(t, err)

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
	plan := ts.Plan("mock")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_1")

	err := ts.Keepers.Projects.CreateAdminProject(ts.Ctx, sub1Addr, plan)
	require.Nil(t, err)

	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	_, err = ts.QueryProjectDeveloper(adm1Addr)
	require.NotNil(t, err)
	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.Nil(t, err)

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
	plan := ts.Plan("mock")

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("pd_adm_1")
	_, dev1Addr := ts.Account("dev1")

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	pk := types.ProjectDeveloperKey(dev1Addr)
	err = ts.TxProjectAddKeys(projectID, adm1Addr, pk)
	require.Nil(t, err)

	spec := common.CreateMockSpec()
	ts.Keepers.Spec.SetSpec(ts.Ctx, spec)

	templates := []struct {
		name                         string
		creator                      string
		projectID                    string
		geolocation                  uint64
		chainPolicies                []planstypes.ChainPolicy
		totalCuLimit                 uint64
		epochCuLimit                 uint64
		maxProvidersToPair           uint64
		setAdminPolicySuccess        bool
		setSubscriptionPolicySuccess bool
	}{
		{
			"valid policy (admin account)", adm1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, false,
		},
		{
			"valid policy (subscription account)", sub1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, true,
		},
		{
			"bad creator (developer account -- not admin)", dev1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			"bad projectID (doesn't exist)", dev1Addr, "fakeProjectId", uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			"invalid geolocation (0)", dev1Addr, projectID, uint64(0),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, false, false,
		},
		{
			// note: currently, we don't verify the chain policies
			"bad chainID (doesn't exist)", sub1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: "LOL",
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"bad API (doesn't exist)", sub1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{"lol"},
			}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"chainID and API not supported (exist in Lava's specs)", sub1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: "ETH1",
				Apis:    []string{"eth_accounts"},
			}},
			100, 10, 3, true, true,
		},
		{
			"epoch CU larger than total CU", sub1Addr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{
				ChainId: spec.Index,
				Apis:    []string{spec.ApiCollections[0].Apis[0].Name},
			}},
			10, 100, 3, false, false,
		},
		{
			"bad maxProvidersToPair", sub1Addr, projectID, uint64(1),
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
				_, err := ts.TxProjectSetPolicy(tt.projectID, tt.creator, newPolicy)
				if tt.setAdminPolicySuccess {
					require.Nil(t, err)
					ts.AdvanceEpoch()
					proj, err := ts.GetProjectForBlock(tt.projectID, ts.BlockHeight())
					require.Nil(t, err)
					require.Equal(t, newPolicy, *proj.AdminPolicy)
				} else {
					require.NotNil(t, err)
				}
			} else {
				_, err := ts.TxProjectSetSubscriptionPolicy(tt.projectID, tt.creator, newPolicy)
				if tt.setSubscriptionPolicySuccess {
					require.Nil(t, err)
					ts.AdvanceEpoch()
					proj, err := ts.GetProjectForBlock(tt.projectID, ts.BlockHeight())
					require.Nil(t, err)
					require.Equal(t, newPolicy, *proj.SubscriptionPolicy)
				} else {
					require.NotNil(t, err)
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
	plan := ts.Plan("mock")

	_, sub1Addr := ts.Account("pd_adm_1")
	_, dev1Addr := ts.Account("dev1")

	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()
	block1 := ts.BlockHeight()

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)
	project, err := ts.GetProjectForBlock(projectID, block1)
	require.Nil(t, err)

	// first epoch to add some delay before adding the developer key.

	ts.AdvanceEpoch()
	block2 := ts.BlockHeight()

	// add developer key (created fixation)
	err = ts.TxProjectAddKeys(project.Index, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Nil(t, err)

	// second epoch to move further, otherwise snapshot will affect the current new
	// dev key instead of creating a separate fixation version.

	ts.AdvanceEpoch()
	block3 := ts.BlockHeight()

	ts.Keepers.Projects.SnapshotSubscriptionProjects(ts.Ctx, sub1Addr)

	// try to charge CUs: should update oldest and second-oldest entries, but not the latest
	// (because the latter is in a new snapshot)

	err = ts.Keepers.Projects.ChargeComputeUnitsToProject(ts.Ctx, project, block1, 1000)
	require.Nil(t, err)

	proj, err := ts.GetProjectForBlock(project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)

	err = ts.Keepers.Projects.ChargeComputeUnitsToProject(ts.Ctx, project, block2, 1000)
	require.Nil(t, err)

	proj, err = ts.GetProjectForBlock(project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(2000), proj.UsedCu)
	proj, err = ts.GetProjectForBlock(project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)
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

	plan := ts.Plan("mock")

	projectData1 := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub1Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err := ts.Keepers.Projects.CreateProject(ts.Ctx, sub1Addr, projectData1, plan)
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        "mockname2",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub2Addr, projectData2, plan)
	require.Nil(t, err)

	ts.AdvanceBlock()

	projectID1 := types.ProjectIndex(sub1Addr, projectData1.Name)
	projectID2 := types.ProjectIndex(sub2Addr, projectData2.Name)

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Nil(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev2Addr))
	require.Nil(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID1, dev2Addr, types.ProjectKey_DEVELOPER))

	res, err := ts.QueryProjectDeveloper(sub1Addr)
	require.Nil(t, err)
	require.Equal(t, 3, len(res.Project.ProjectKeys))

	// add twice - ok
	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// del twice - fail
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NotNil(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.NotNil(t, err)

	// del takes effect in next epoch
	ts.AdvanceEpoch()

	require.False(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// add, del, and add again in same epoch
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)

	err = ts.TxProjectAddKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.Nil(t, err)
	err = ts.TxProjectAddKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.NotNil(t, err)

	ts.AdvanceEpoch()

	// add, del admin (admin should invalid immediately)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.Nil(t, err)
	err = ts.TxProjectAddKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.NotNil(t, err)

	// del takes effect in next epoch
	ts.AdvanceEpoch()

	require.True(t, ts.isKeyInProject(projectID2, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID2, dev4Addr, types.ProjectKey_DEVELOPER))
	require.False(t, ts.isKeyInProject(projectID2, adm2Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID2, dev6Addr, types.ProjectKey_DEVELOPER))

	// add dev to two projects in same epoch (latter fails)
	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectDeveloperKey(dev5Addr))
	require.Nil(t, err)
	err = ts.TxProjectAddKeys(projectID2, sub1Addr, types.ProjectDeveloperKey(dev5Addr))
	require.NotNil(t, err)
}

func TestDelKeysDelProjectSameEpoch(t *testing.T) {
	var err error

	ts := newTester(t)
	ts.SetupAccounts(1, 1, 2) // 1 sub, 1 adm, 2 dev

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("adm1")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")

	plan := ts.Plan("mock")

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
		require.Nil(t, err)
	}

	ts.AdvanceEpoch()

	// part (1): delete keys then project

	// delete key from each project
	err = ts.TxProjectDelKeys(projectsID[0], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.TxProjectDelKeys(projectsID[1], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Nil(t, err)

	// now delete the projects (double delete) in same epoch
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[0])
	require.Nil(t, err)
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[1])
	require.Nil(t, err)

	proj, err := ts.GetProjectForBlock(projectsID[0], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.GetProjectForBlock(projectsID[1], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch()

	proj, err = ts.GetProjectForBlock(projectsID[0], ts.BlockHeight())
	require.NotNil(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[1], ts.BlockHeight())
	require.NotNil(t, err)

	// should not panic
	ts.AdvanceBlocks(2 * commontypes.STALE_ENTRY_TIME)

	// part (2): delete project then keys

	// delete the projects
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[2])
	require.Nil(t, err)
	err = ts.Keepers.Projects.DeleteProject(ts.Ctx, sub1Addr, projectsID[3])
	require.Nil(t, err)

	// delete key from each project: should being  (project being deleted)
	err = ts.TxProjectDelKeys(projectsID[2], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NotNil(t, err)
	err = ts.TxProjectDelKeys(projectsID[3], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NotNil(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[2], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.GetProjectForBlock(projectsID[3], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch()

	proj, err = ts.GetProjectForBlock(projectsID[2], ts.BlockHeight())
	require.NotNil(t, err)

	proj, err = ts.GetProjectForBlock(projectsID[3], ts.BlockHeight())
	require.NotNil(t, err)

	// should not panic
	ts.AdvanceBlocks(2 * commontypes.STALE_ENTRY_TIME)
}

func TestAddDevKeyToDifferentProjectsInSameBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 0, 1) // 2 sub, 0 adm, 1 dev

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")
	_, dev1Addr := ts.Account("dev1")

	plan := ts.Plan("mock")

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
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        projectName2,
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.Keepers.Projects.CreateProject(ts.Ctx, sub2Addr, projectData2, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	err = ts.TxProjectAddKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Nil(t, err)

	err = ts.TxProjectAddKeys(projectID2, sub2Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NotNil(t, err) // developer was already added to the first project

	ts.AdvanceEpoch()

	res1, err := ts.QueryProjectDeveloper(sub1Addr)
	require.Nil(t, err)
	res2, err := ts.QueryProjectDeveloper(sub2Addr)
	require.Nil(t, err)

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

			plan := ts.Plan("mock")
			plan.PlanPolicy.SelectedProvidersMode = tt.planMode
			plan.PlanPolicy.SelectedProviders = providersSet.planProviders

			err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{plan})
			if tt.planPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
			require.Nil(t, err)

			res, err := ts.QuerySubscriptionListProjects(sub1Addr)
			require.Nil(t, err)
			require.Equal(t, 1, len(res.Projects))

			admProject, err := ts.GetProjectForBlock(res.Projects[0], ts.BlockHeight())
			require.Nil(t, err)

			policy.SelectedProvidersMode = tt.projMode
			policy.SelectedProviders = providersSet.projProviders

			_, err = ts.TxProjectSetPolicy(admProject.Index, sub1Addr, policy)
			if tt.projPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

			policy.SelectedProvidersMode = tt.subMode
			policy.SelectedProviders = providersSet.subProviders

			_, err = ts.TxProjectSetSubscriptionPolicy(admProject.Index, sub1Addr, policy)
			if tt.subPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
