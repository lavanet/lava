package keeper_test

import (
	"context"
	"math"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

const projectName = "mockname"

type testStruct struct {
	t        *testing.T
	servers  *testkeeper.Servers
	keepers  *testkeeper.Keepers
	_ctx     context.Context
	ctx      sdk.Context
	accounts map[string]string
	projects map[string]types.ProjectData
}

func newTestStruct(t *testing.T) *testStruct {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ts := &testStruct{
		t:       t,
		servers: servers,
		keepers: keepers,
		_ctx:    _ctx,
		ctx:     sdk.UnwrapSDKContext(_ctx),
	}
	ts.AdvanceEpoch(1)
	return ts
}

func (ts *testStruct) prepareData(numSub, numAdmin, numDevel int) {
	ts.accounts = make(map[string]string)
	for i := 0; i < numSub; i++ {
		k := "sub" + strconv.Itoa(i+1)
		v := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
		ts.accounts[k] = v
	}
	for i := 0; i < numAdmin; i++ {
		k := "adm" + strconv.Itoa(i+1)
		v := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
		ts.accounts[k] = v
	}
	for i := 0; i < numDevel; i++ {
		k := "dev" + strconv.Itoa(i+1)
		v := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
		ts.accounts[k] = v
	}

	ts.projects = make(map[string]types.ProjectData)

	ts.accounts["pd1_adm"] = common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
	ts.accounts["pd2_both"] = common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
	ts.accounts["pd3_adm"] = common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
	ts.accounts["pd3_dev"] = common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()

	// admin key
	keys_1_admin := []types.ProjectKey{
		types.ProjectAdminKey(ts.accounts["pd1_adm"]),
	}
	// developer key
	keys_1_admin_dev := []types.ProjectKey{
		types.NewProjectKey(ts.accounts["pd2_both"]).
			AddType(types.ProjectKey_ADMIN).
			AddType(types.ProjectKey_DEVELOPER),
	}

	// both (admin+developer) key
	keys_2_admin_and_dev := []types.ProjectKey{
		types.ProjectAdminKey(ts.accounts["pd3_adm"]),
		types.ProjectDeveloperKey(ts.accounts["pd3_dev"]),
	}

	policy1 := &planstypes.Policy{
		GeolocationProfile: math.MaxUint64,
		MaxProvidersToPair: 2,
	}

	templates := []struct {
		code    string
		name    string
		enabled bool
		keys    []types.ProjectKey
		policy  *planstypes.Policy
	}{
		// project with admin key, enabled, has policy
		{"pd1", "mock_project_1", true, keys_1_admin, policy1},
		// project with "both" key, disabled, with policy
		{"pd2", "mock_project_2", false, keys_1_admin_dev, policy1},
		// project with 2 keys (one admin, one developer) disabled, no policy
		{"pd3", "mock_project_3", false, keys_2_admin_and_dev, nil},
	}

	for _, tt := range templates {
		ts.projects[tt.code] = types.ProjectData{
			Name:        tt.name,
			Enabled:     tt.enabled,
			ProjectKeys: tt.keys,
			Policy:      tt.policy,
		}
	}
}

func (ts *testStruct) BlockHeight() uint64 {
	return uint64(ts.ctx.BlockHeight())
}

func (ts *testStruct) AdvanceBlock(count int) {
	for i := 0; i < count; i += 1 {
		ts._ctx = testkeeper.AdvanceBlock(ts._ctx, ts.keepers)
	}
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func (ts *testStruct) AdvanceEpoch(count int) {
	for i := 0; i < count; i += 1 {
		ts._ctx = testkeeper.AdvanceEpoch(ts._ctx, ts.keepers)
	}
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func (ts *testStruct) isKeyInProject(index, key string, kind types.ProjectKey_Type) bool {
	resp, err := ts.keepers.Projects.Info(ts._ctx, &types.QueryInfoRequest{Project: index})
	require.Nil(ts.t, err, "project: "+index+", key: "+key)
	pk := resp.Project.GetKey(key)
	return pk.IsType(kind)
}

func (ts *testStruct) addProjectKeys(index, creator string, projectKeys ...types.ProjectKey) error {
	msg := types.MsgAddKeys{
		Creator:     creator,
		Project:     index,
		ProjectKeys: projectKeys,
	}
	_, err := ts.servers.ProjectServer.AddKeys(ts._ctx, &msg)
	return err
}

func (ts *testStruct) delProjectKeys(index, creator string, projectKeys ...types.ProjectKey) error {
	msg := types.MsgDelKeys{
		Creator:     creator,
		Project:     index,
		ProjectKeys: projectKeys,
	}
	_, err := ts.servers.ProjectServer.DelKeys(ts._ctx, &msg)
	return err
}

func TestCreateDefaultProject(t *testing.T) {
	ts := newTestStruct(t)

	subAddr := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
	plan := common.CreateMockPlan()

	err := ts.keepers.Projects.CreateAdminProject(ts.ctx, subAddr, plan)
	require.Nil(t, err)

	// subscription key is a developer in the default project
	response1, err := ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	response2, err := ts.keepers.Projects.Info(ts._ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)
}

func TestCreateProject(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(1, 0, 0) // 1 sub, 0 adm, 0 dev

	projectData := ts.projects["pd2"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["sub1"]
	admAddr := ts.accounts["pd2_both"]

	err := ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	// test invalid project name
	defaultProjectName := types.ADMIN_PROJECT_NAME
	longProjectName := strings.Repeat(defaultProjectName, types.MAX_PROJECT_NAME_LEN+1)
	invalidProjectName := "projectName,"

	testProjectData := projectData
	testProjectData.ProjectKeys = []types.ProjectKey{}

	nameTests := []struct {
		name        string
		projectName string
	}{
		{"bad projectName (duplicate)", projectData.Name},
		{"bad projectName (too long)", longProjectName},
		{"bad projectName (contains comma)", invalidProjectName},
		{"bad projectName (empty)", ""},
	}

	for _, tt := range nameTests {
		t.Run(tt.name, func(t *testing.T) {
			testProjectData.Name = tt.projectName
			err = ts.keepers.Projects.CreateProject(ts.ctx, subAddr, testProjectData, plan)
			require.NotNil(t, err)
		})
	}

	// continue testing traits that are not related to the project's name
	// try creating a project with invalid project keys
	invalidKeysProjectData := projectData
	invalidKeysProjectData.Name = "nonDuplicateProjectName"
	invalidKeysProjectData.ProjectKeys = []types.ProjectKey{
		types.ProjectDeveloperKey(subAddr),
		types.ProjectAdminKey(subAddr).AddType(0x4),
	}

	// should fail because there's an invalid key
	_, err = ts.servers.SubscriptionServer.AddProject(ts._ctx, &subscriptiontypes.MsgAddProject{
		Creator:     subAddr,
		ProjectData: invalidKeysProjectData,
	})
	require.NotNil(t, err)

	// subscription key is not a developer, so get project by developer should fail (if it succeeds,
	// then the valid project key from invalidKeysProjectData was registered, which is undesired!)
	_, err = ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.NotNil(t, err)

	response1, err := ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: admAddr})
	require.Nil(t, err)

	response2, err := ts.keepers.Projects.Info(ts._ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)

	_, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, response1.Project.Index, ts.BlockHeight())
	require.Nil(t, err)

	// there should be one project key
	require.Equal(t, 1, len(response2.Project.ProjectKeys))

	// the project key is the admin key
	require.Equal(t, response2.Project.ProjectKeys[0].Key, admAddr)

	// the admin is both an admin and a developer
	require.True(t, response2.Project.ProjectKeys[0].IsType(types.ProjectKey_ADMIN))
	require.True(t, response2.Project.ProjectKeys[0].IsType(types.ProjectKey_DEVELOPER))
}

func TestProjectsServerAPI(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(2, 1, 5) // 2 sub, 1 adm, 5 dev

	sub1Acc := common.CreateNewAccount(ts._ctx, *ts.keepers, 20000)
	sub1Addr := sub1Acc.Addr.String()

	dev1Addr := ts.accounts["dev1"]
	dev2Addr := ts.accounts["dev2"]

	plan := common.CreateMockPlan()
	ts.keepers.Plans.AddPlan(ts.ctx, plan)

	common.BuySubscription(t, ts._ctx, *ts.keepers, *ts.servers, sub1Acc, plan.Index)

	projectData := types.ProjectData{
		Name:        "mockname2",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)},
		Policy:      &plan.PlanPolicy,
	}

	projectID := types.ProjectIndex(sub1Addr, projectData.Name)

	msgAddProject := &subscriptiontypes.MsgAddProject{
		Creator:     sub1Addr,
		ProjectData: projectData,
	}
	_, err := ts.servers.SubscriptionServer.AddProject(ts._ctx, msgAddProject)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	msgAddKeys := types.MsgAddKeys{
		Creator:     sub1Addr,
		Project:     projectID,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(dev2Addr)},
	}
	_, err = ts.servers.ProjectServer.AddKeys(ts._ctx, &msgAddKeys)
	require.Nil(t, err)

	ts.AdvanceBlock(1)

	require.True(t, ts.isKeyInProject(projectID, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID, dev2Addr, types.ProjectKey_DEVELOPER))

	msgQueryDev := &types.QueryDeveloperRequest{Developer: sub1Addr}
	res, err := ts.keepers.Projects.Developer(ts._ctx, msgQueryDev)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Project.ProjectKeys))
}

func TestDeleteProject(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(1, 0, 0) // 1 sub, 0 adm, 0 dev

	projectData := ts.projects["pd2"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["sub1"]
	projectID := types.ProjectIndex(subAddr, projectData.Name)

	err := ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	_, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectID, ts.BlockHeight())
	require.Nil(t, err)

	err = ts.keepers.Projects.DeleteProject(ts.ctx, subAddr, "nonsense")
	require.NotNil(t, err)

	err = ts.keepers.Projects.DeleteProject(ts.ctx, subAddr, projectID)
	require.Nil(t, err)

	_, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectID, ts.BlockHeight())
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	_, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectID, ts.BlockHeight())
	require.NotNil(t, err)
}

func TestAddDelKeys(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(1, 0, 2) // 1 sub, 0 adm, 2 dev

	projectData := ts.projects["pd3"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["sub1"]
	admAddr := ts.accounts["pd3_adm"]
	dev1Addr := ts.accounts["pd3_dev"]
	dev2Addr := ts.accounts["dev1"]
	dev3Addr := ts.accounts["dev2"]

	err := ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	projectRes, err := ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: dev1Addr})
	require.Nil(t, err)

	project := projectRes.Project
	pk := types.ProjectAdminKey(dev1Addr)

	// try adding myself as admin, should fail
	err = ts.addProjectKeys(project.Index, dev1Addr, pk)
	require.NotNil(t, err)

	// admin key adding an invalid key
	pk = types.NewProjectKey(dev2Addr).AddType(0x4)
	err = ts.addProjectKeys(project.Index, admAddr, pk)
	require.NotNil(t, err)

	// admin key adding a developer
	pk = types.ProjectDeveloperKey(dev2Addr)
	err = ts.addProjectKeys(project.Index, admAddr, pk)
	require.Nil(t, err)

	// developer tries to add the second developer as admin
	pk = types.ProjectAdminKey(dev2Addr)
	err = ts.addProjectKeys(project.Index, dev1Addr, pk)
	require.NotNil(t, err)

	// admin adding admin
	pk = types.ProjectAdminKey(dev1Addr)
	err = ts.addProjectKeys(project.Index, admAddr, pk)
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(project.Index, dev2Addr, types.ProjectKey_DEVELOPER))

	ts.AdvanceEpoch(1)

	// new admin adding another developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.addProjectKeys(project.Index, dev1Addr, pk)
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))

	// fetch project with new developer
	_, err = ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: dev3Addr})
	require.Nil(t, err)

	// developer delete admin
	pk = types.ProjectAdminKey(admAddr)
	err = ts.delProjectKeys(project.Index, dev2Addr, pk)
	require.NotNil(t, err)

	// developer delete developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.delProjectKeys(project.Index, dev2Addr, pk)
	require.NotNil(t, err)

	// new admin delete subscription owner (admin)
	pk = types.ProjectAdminKey(subAddr)
	err = ts.delProjectKeys(project.Index, admAddr, pk)
	require.NotNil(t, err)

	// subscription owner (admin) delete other admin
	pk = types.ProjectAdminKey(admAddr)
	err = ts.delProjectKeys(project.Index, subAddr, pk)
	require.Nil(t, err)

	// admin delete developer (admin removed already!)
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.delProjectKeys(project.Index, admAddr, pk)
	require.NotNil(t, err)

	// subscription owner (admin) delete developer
	pk = types.ProjectDeveloperKey(dev3Addr)
	err = ts.delProjectKeys(project.Index, subAddr, pk)
	require.Nil(t, err)

	// deletion take effect in next epoch
	require.True(t, ts.isKeyInProject(project.Index, admAddr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))
	ts.AdvanceEpoch(1)
	require.False(t, ts.isKeyInProject(project.Index, admAddr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(project.Index, dev3Addr, types.ProjectKey_DEVELOPER))
}

func TestAddAdminInTwoProjects(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(1, 0, 0) // 1 sub, 0 admin, 0 devel

	projectData := ts.projects["pd1"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["sub1"]
	admAddr := ts.accounts["pd1_adm"]

	err := ts.keepers.Projects.CreateAdminProject(ts.ctx, subAddr, plan)
	require.Nil(t, err)

	// this is not supposed to fail because you can use the same admin key for two different projects
	// creating a regular project (not admin project) so subAccount won't be a developer there
	err = ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	_, err = ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: admAddr})
	require.NotNil(t, err)

	response, err := ts.keepers.Projects.Developer(ts._ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.Nil(t, err)
	require.Equal(t, response.Project.Index, types.ProjectIndex(subAddr, types.ADMIN_PROJECT_NAME))
}

func TestSetPolicy(t *testing.T) {
	setPolicyTest(t, true)
}

func TestSetSubscriptionPolicy(t *testing.T) {
	setPolicyTest(t, false)
}

func setPolicyTest(t *testing.T, testAdminPolicy bool) {
	ts := newTestStruct(t)
	ts.prepareData(1, 0, 1) // 1 sub, 0 admin, 1 data

	projectData := ts.projects["pd1"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["sub1"]
	admAddr := ts.accounts["pd1_adm"]
	devAddr := ts.accounts["dev1"]

	projectID := types.ProjectIndex(subAddr, projectData.Name)

	err := ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	pk := types.ProjectDeveloperKey(devAddr)
	err = ts.keepers.Projects.AddKeysToProject(ts.ctx, projectID, admAddr, []types.ProjectKey{pk})
	require.Nil(t, err)

	spec := common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(ts.ctx, spec)

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
			"valid policy (admin account)", admAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, true, false,
		},

		{
			"valid policy (subscription account)", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, true, true,
		},

		{
			"bad creator (developer account -- not admin)", devAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			"bad projectID (doesn't exist)", devAddr, "fakeProjectId", uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			"invalid geolocation (0)", devAddr, projectID, uint64(0),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			// note: currently, we don't verify the chain policies
			"bad chainID (doesn't exist)", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: "LOL", Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			100, 10, 3, true, true,
		},

		{
			// note: currently, we don't verify the chain policies
			"bad API (doesn't exist)", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{"lol"}}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"chainID and API not supported (exist in Lava's specs)", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: "ETH1", Apis: []string{"eth_accounts"}}},
			100, 10, 3, true, true,
		},
		{
			"epoch CU larger than total CU", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
			10, 100, 3, false, false,
		},
		{
			"bad maxProvidersToPair", subAddr, projectID, uint64(1),
			[]planstypes.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.ApiCollections[0].Apis[0].Name}}},
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
				SetPolicyMessage := types.MsgSetPolicy{
					Creator: tt.creator,
					Policy:  newPolicy,
					Project: tt.projectID,
				}

				err = SetPolicyMessage.ValidateBasic()
				require.Nil(t, err)

				_, err := ts.servers.ProjectServer.SetPolicy(ts._ctx, &SetPolicyMessage)
				if tt.setAdminPolicySuccess {
					require.Nil(t, err)
					ts.AdvanceEpoch(1)

					proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, tt.projectID, ts.BlockHeight())
					require.Nil(t, err)

					require.Equal(t, newPolicy, *proj.AdminPolicy)
				} else {
					require.NotNil(t, err)
				}
			} else {
				setSubscriptionPolicyMessage := types.MsgSetSubscriptionPolicy{
					Creator:  tt.creator,
					Policy:   newPolicy,
					Projects: []string{tt.projectID},
				}

				err = setSubscriptionPolicyMessage.ValidateBasic()
				require.Nil(t, err)

				_, err := ts.servers.ProjectServer.SetSubscriptionPolicy(ts._ctx, &setSubscriptionPolicyMessage)
				if tt.setSubscriptionPolicySuccess {
					require.Nil(t, err)
					ts.AdvanceEpoch(1)

					proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, tt.projectID, ts.BlockHeight())
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
	ts := newTestStruct(t)
	ts.prepareData(0, 0, 1) // 0 sub, 0 adm, 1 dev

	projectData := ts.projects["pd1"]
	plan := common.CreateMockPlan()

	subAddr := ts.accounts["pd1_adm"]
	devAddr := ts.accounts["dev1"]

	err := ts.keepers.Projects.CreateProject(ts.ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)
	block1 := ts.BlockHeight()

	projectID := types.ProjectIndex(subAddr, projectData.Name)
	project, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectID, block1)
	require.Nil(t, err)

	// first epoch to add some delay before adding the developer key.

	ts.AdvanceEpoch(1)
	block2 := ts.BlockHeight()

	// add developer key (created fixation)
	err = ts.addProjectKeys(project.Index, subAddr, types.ProjectDeveloperKey(devAddr))
	require.Nil(t, err)

	// second epoch to move further, otherwise snapshot will affect the current new
	// dev key instead of creating a separate fixation version.

	ts.AdvanceEpoch(1)
	block3 := ts.BlockHeight()

	ts.keepers.Projects.SnapshotSubscriptionProjects(ts.ctx, subAddr)

	// try to charge CUs: should update oldest and second-oldest entries, but not the latest
	// (because the latter is in a new snapshot)

	err = ts.keepers.Projects.ChargeComputeUnitsToProject(ts.ctx, project, block1, 1000)
	require.Nil(t, err)

	proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)

	ts.keepers.Projects.ChargeComputeUnitsToProject(ts.ctx, project, block2, 1000)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(2000), proj.UsedCu)
	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)
}

func TestAddDelKeysSameEpoch(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(2, 2, 6) // 2 sub, 1 adm, 5 dev

	sub1Addr := ts.accounts["sub1"]
	sub2Addr := ts.accounts["sub2"]
	adm1Addr := ts.accounts["adm1"]
	adm2Addr := ts.accounts["adm2"]
	dev1Addr := ts.accounts["dev1"]
	dev2Addr := ts.accounts["dev2"]
	dev3Addr := ts.accounts["dev3"]
	dev4Addr := ts.accounts["dev4"]
	dev5Addr := ts.accounts["dev5"]
	dev6Addr := ts.accounts["dev6"]

	plan := common.CreateMockPlan()

	projectData1 := types.ProjectData{
		Name:        "mockname1",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub1Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err := ts.keepers.Projects.CreateProject(ts.ctx, sub1Addr, projectData1, plan)
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        "mockname2",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.keepers.Projects.CreateProject(ts.ctx, sub2Addr, projectData2, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	projectID1 := types.ProjectIndex(sub1Addr, projectData1.Name)
	projectID2 := types.ProjectIndex(sub2Addr, projectData2.Name)

	err = ts.keepers.Projects.AddKeysToProject(ts.ctx, projectID1, sub1Addr,
		[]types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)})
	require.Nil(t, err)
	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))

	ts.AdvanceBlock(1)

	err = ts.keepers.Projects.AddKeysToProject(ts.ctx, projectID1, sub1Addr,
		[]types.ProjectKey{types.ProjectDeveloperKey(dev2Addr)})
	require.Nil(t, err)

	require.True(t, ts.isKeyInProject(projectID1, dev1Addr, types.ProjectKey_DEVELOPER))
	require.True(t, ts.isKeyInProject(projectID1, dev2Addr, types.ProjectKey_DEVELOPER))

	proj, err := ts.keepers.Projects.GetProjectForDeveloper(ts.ctx, sub1Addr, ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 3, len(proj.ProjectKeys))

	// add twice - ok
	err = ts.addProjectKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.addProjectKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.Nil(t, err)

	ts.AdvanceEpoch(1)
	require.True(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// del twice - fail
	err = ts.delProjectKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID1, sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NotNil(t, err)
	err = ts.delProjectKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID1, sub1Addr, types.ProjectDeveloperKey(dev3Addr))
	require.NotNil(t, err)

	ts.AdvanceEpoch(1)
	require.False(t, ts.isKeyInProject(projectID1, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID1, dev3Addr, types.ProjectKey_DEVELOPER))

	// add, del (and add again) in same epoch
	err = ts.addProjectKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.addProjectKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)

	err = ts.addProjectKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.Nil(t, err)
	err = ts.addProjectKeys(projectID2, adm1Addr, types.ProjectDeveloperKey(dev4Addr))
	require.NotNil(t, err)

	// add, del: admin, should be invalid immediately
	err = ts.addProjectKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.Nil(t, err)
	err = ts.addProjectKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID2, sub2Addr, types.ProjectAdminKey(adm2Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectID2, adm2Addr, types.ProjectDeveloperKey(dev6Addr))
	require.NotNil(t, err)

	ts.AdvanceEpoch(1)
	require.True(t, ts.isKeyInProject(projectID2, adm1Addr, types.ProjectKey_ADMIN))
	require.False(t, ts.isKeyInProject(projectID2, dev4Addr, types.ProjectKey_DEVELOPER))
	require.False(t, ts.isKeyInProject(projectID2, adm2Addr, types.ProjectKey_ADMIN))
	require.True(t, ts.isKeyInProject(projectID2, dev6Addr, types.ProjectKey_DEVELOPER))

	// add dev to two projects in same epoch - latter fails
	err = ts.addProjectKeys(projectID2, sub2Addr, types.ProjectDeveloperKey(dev5Addr))
	require.Nil(t, err)
	err = ts.addProjectKeys(projectID2, sub1Addr, types.ProjectDeveloperKey(dev5Addr))
	require.NotNil(t, err)
}

func TestDelKeysDelProjectSameEpoch(t *testing.T) {
	var err error

	ts := newTestStruct(t)
	ts.prepareData(1, 1, 2) // 1 sub, 1 adm, 1 dev

	sub1Addr := ts.accounts["sub1"]
	adm1Addr := ts.accounts["adm1"]
	dev1Addr := ts.accounts["dev1"]
	dev2Addr := ts.accounts["dev2"]

	plan := common.CreateMockPlan()

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
		err = ts.keepers.Projects.CreateProject(ts.ctx, sub1Addr, projectsData[i], plan)
		require.Nil(t, err)
	}

	ts.AdvanceEpoch(1)

	// part (1): delete keys then project

	// delete key from each project
	err = ts.delProjectKeys(projectsID[0], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.Nil(t, err)
	err = ts.delProjectKeys(projectsID[1], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.Nil(t, err)

	// now delete the projects (double delete) in same epoch
	err = ts.keepers.Projects.DeleteProject(ts.ctx, sub1Addr, projectsID[0])
	require.Nil(t, err)
	err = ts.keepers.Projects.DeleteProject(ts.ctx, sub1Addr, projectsID[1])
	require.Nil(t, err)

	proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[0], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[1], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch(1)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[0], ts.BlockHeight())
	require.NotNil(t, err)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[1], ts.BlockHeight())
	require.NotNil(t, err)

	// should not panic
	ts.AdvanceBlock(2 * int(commontypes.STALE_ENTRY_TIME))

	// part (2): delete project then keys

	// delete the projects
	err = ts.keepers.Projects.DeleteProject(ts.ctx, sub1Addr, projectsID[2])
	require.Nil(t, err)
	err = ts.keepers.Projects.DeleteProject(ts.ctx, sub1Addr, projectsID[3])
	require.Nil(t, err)

	// delete key from each project: should being  (project being deleted)
	err = ts.delProjectKeys(projectsID[2], sub1Addr, types.ProjectAdminKey(adm1Addr))
	require.NotNil(t, err)
	err = ts.delProjectKeys(projectsID[3], sub1Addr, types.ProjectDeveloperKey(dev1Addr))
	require.NotNil(t, err)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[2], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[3], ts.BlockHeight())
	require.Nil(t, err)
	require.Equal(t, 1, len(proj.ProjectKeys))

	// wait for next epoch for delete(s) to take effect
	ts.AdvanceEpoch(1)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[2], ts.BlockHeight())
	require.NotNil(t, err)

	proj, err = ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectsID[3], ts.BlockHeight())
	require.NotNil(t, err)

	// should not panic
	ts.AdvanceBlock(2 * int(commontypes.STALE_ENTRY_TIME))
}

func TestAddDevKeyToDifferentProjectsInSameBlock(t *testing.T) {
	ts := newTestStruct(t)
	ts.prepareData(2, 0, 1) // 2 sub, 0 adm, 1 dev

	sub1Addr := ts.accounts["sub1"]
	sub2Addr := ts.accounts["sub2"]
	dev1Addr := ts.accounts["dev1"]

	plan := common.CreateMockPlan()

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
	err := ts.keepers.Projects.CreateProject(ts.ctx, sub1Addr, projectData1, plan)
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        projectName2,
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(sub2Addr)},
		Policy:      &plan.PlanPolicy,
	}
	err = ts.keepers.Projects.CreateProject(ts.ctx, sub2Addr, projectData2, plan)
	require.Nil(t, err)

	ts.AdvanceEpoch(1)

	err = ts.keepers.Projects.AddKeysToProject(ts.ctx, projectID1, sub1Addr,
		[]types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)})
	require.Nil(t, err)

	err = ts.keepers.Projects.AddKeysToProject(ts.ctx, projectID2, sub2Addr,
		[]types.ProjectKey{types.ProjectDeveloperKey(dev1Addr)})
	require.NotNil(t, err) // developer was already added to the first project

	ts.AdvanceEpoch(1)

	proj1, err := ts.keepers.Projects.GetProjectForDeveloper(ts.ctx, sub1Addr, ts.BlockHeight())
	require.Nil(t, err)

	proj2, err := ts.keepers.Projects.GetProjectForDeveloper(ts.ctx, sub2Addr, ts.BlockHeight())
	require.Nil(t, err)

	require.Equal(t, 2, len(proj1.ProjectKeys))
	require.Equal(t, 1, len(proj2.ProjectKeys))
}

func TestSetPolicySelectedProviders(t *testing.T) {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	adm1Addr := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()
	projectData := types.ProjectData{
		Name:        "name",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{Key: adm1Addr, Kinds: uint32(types.ProjectKey_ADMIN)}},
		Policy:      &planstypes.Policy{MaxProvidersToPair: 2, GeolocationProfile: math.MaxUint64},
	}
	subAddr := projectData.ProjectKeys[0].Key
	projPolicy := projectData.Policy

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
		{[]string{subAddr}, []string{subAddr}, []string{subAddr}},
		{[]string{subAddr}, []string{subAddr}, []string{}},
		{[]string{"lalala"}, []string{"lalala"}, []string{"lalala"}},
		{[]string{subAddr, subAddr}, []string{subAddr, subAddr}, []string{subAddr, subAddr}},
		{[]string{subAddr}, []string{}, []string{}},
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
			plan := common.CreateMockPlan()

			plan.PlanPolicy.SelectedProvidersMode = tt.planMode
			plan.PlanPolicy.SelectedProviders = providersSet.planProviders

			err := testkeeper.SimulatePlansAddProposal(ctx, keepers.Plans, []planstypes.Plan{plan})
			if tt.planPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

			_, err = servers.SubscriptionServer.Buy(_ctx, &subscriptiontypes.MsgBuy{
				Creator:  subAddr,
				Consumer: subAddr,
				Index:    plan.Index,
				Duration: 1,
			})
			require.Nil(t, err)

			subProjects, err := keepers.Subscription.ListProjects(_ctx, &subscriptiontypes.QueryListProjectsRequest{
				Subscription: subAddr,
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(subProjects.Projects))

			adminProject, err := keepers.Projects.GetProjectForBlock(ctx, subProjects.Projects[0], uint64(ctx.BlockHeight()))
			require.Nil(t, err)

			projPolicy.SelectedProvidersMode = tt.projMode
			projPolicy.SelectedProviders = providersSet.projProviders

			_, err = servers.ProjectServer.SetPolicy(_ctx, &types.MsgSetPolicy{
				Creator: subAddr,
				Project: adminProject.Index,
				Policy:  *projPolicy,
			})
			if tt.projPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

			projPolicy.SelectedProvidersMode = tt.subMode
			projPolicy.SelectedProviders = providersSet.subProviders

			_, err = servers.ProjectServer.SetSubscriptionPolicy(_ctx, &types.MsgSetSubscriptionPolicy{
				Creator:  subAddr,
				Projects: []string{adminProject.Index},
				Policy:   *projPolicy,
			})
			if tt.subPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestSetPolicyByGeolocation(t *testing.T) {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	// for convinience
	GLS := uint64(planstypes.Geolocation_value["GLS"])
	GL := uint64(planstypes.Geolocation_value["GL"])
	USE := uint64(planstypes.Geolocation_value["USE"])
	EU := uint64(planstypes.Geolocation_value["EU"])
	USE_EU := USE + EU

	// propose all plans
	freePlan := planstypes.Plan{
		Index: "free",
		Block: uint64(ctx.BlockHeight()),
		Price: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 4, // USE
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	basicPlan := planstypes.Plan{
		Index: "basic",
		Block: uint64(ctx.BlockHeight()),
		Price: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 0, // GLS
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	premiumPlan := planstypes.Plan{
		Index: "premium",
		Block: uint64(ctx.BlockHeight()),
		Price: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: planstypes.Policy{
			GeolocationProfile: 65535, // GL
			TotalCuLimit:       10,
			EpochCuLimit:       2,
			MaxProvidersToPair: 2,
		},
	}

	plans := []planstypes.Plan{freePlan, basicPlan, premiumPlan}
	err := testkeeper.SimulatePlansAddProposal(ctx, keepers.Plans, plans)
	require.Nil(t, err)

	freeUser := common.CreateNewAccount(_ctx, *keepers, 10000)
	basicUser := common.CreateNewAccount(_ctx, *keepers, 10000)
	premiumUser := common.CreateNewAccount(_ctx, *keepers, 10000)

	common.BuySubscription(t, _ctx, *keepers, *servers, freeUser, freePlan.Index)
	common.BuySubscription(t, _ctx, *keepers, *servers, basicUser, basicPlan.Index)
	common.BuySubscription(t, _ctx, *keepers, *servers, premiumUser, premiumPlan.Index)

	templates := []struct {
		name           string
		dev            common.Account
		planIndex      int
		setGeo         uint64
		expectedGeo    uint64
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
			require.Nil(t, err)

			projIndex := devResponse.Project.Index

			_, err = servers.ProjectServer.SetPolicy(_ctx, &types.MsgSetPolicy{
				Creator: tt.dev.Addr.String(),
				Project: projIndex,
				Policy: planstypes.Policy{
					GeolocationProfile: tt.setGeo,
					TotalCuLimit:       10,
					EpochCuLimit:       2,
					MaxProvidersToPair: 2,
				},
			})
			if tt.setPolicyValid {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
				return
			}
			_ctx = testkeeper.AdvanceEpoch(_ctx, keepers) // apply the new policy

			devResponse, err = keepers.Projects.Developer(_ctx, &types.QueryDeveloperRequest{
				Developer: tt.dev.Addr.String(),
			})
			require.Nil(t, err)

			policies := []*planstypes.Policy{
				&plans[tt.planIndex].PlanPolicy,
				devResponse.Project.AdminPolicy,
				devResponse.Project.SubscriptionPolicy,
			}
			strictestGeo, err := keepers.Pairing.CalculateEffectiveGeolocationFromPolicies(policies)
			if !tt.badGeo {
				require.Nil(t, err)
				require.Equal(t, tt.expectedGeo, strictestGeo)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
