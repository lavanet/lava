package keeper_test

import (
	"context"
	"math"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/projects/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

const projectName = "mockname"

func prepareProjectsData(ctx context.Context, keepers *testkeeper.Keepers) (projects []types.ProjectData) {
	adm1Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	adm2Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	adm3Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	dev3Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()

	typeAdmin := []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}
	typeDevel := []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}
	typeBoth := []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN, types.ProjectKey_DEVELOPER}

	// admin key
	keys_1_admin := []types.ProjectKey{
		{Key: adm1Addr, Types: typeAdmin},
	}
	// developer key
	keys_1_admin_dev := []types.ProjectKey{
		{Key: adm2Addr, Types: typeBoth},
	}
	// both (admin+developer) key
	keys_2_admin_and_dev := []types.ProjectKey{
		{Key: adm3Addr, Types: typeAdmin},
		{Key: dev3Addr, Types: typeDevel},
	}

	policy1 := &types.Policy{
		GeolocationProfile: math.MaxUint64,
		MaxProvidersToPair: 2,
	}

	templates := []struct {
		name    string
		enabled bool
		keys    []types.ProjectKey
		policy  *types.Policy
	}{
		// project with admin key, enabled, has policy
		{"mock_project_1", true, keys_1_admin, policy1},
		// project with "both" key, disabled, with policy
		{"mock_project_2", false, keys_1_admin_dev, policy1},
		// project with 2 keys (one admin, one developer) disabled, no policy
		{"mock_project_3", false, keys_2_admin_and_dev, nil},
	}

	for _, tt := range templates {
		projectData := types.ProjectData{
			Name:        tt.name,
			Description: "",
			Enabled:     tt.enabled,
			ProjectKeys: tt.keys,
			Policy:      tt.policy,
		}
		projects = append(projects, projectData)
	}

	return projects
}

func TestCreateDefaultProject(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)

	subAddr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	plan := common.CreateMockPlan()

	err := keepers.Projects.CreateAdminProject(sdk.UnwrapSDKContext(ctx), subAddr, plan)
	require.Nil(t, err)

	// subscription key is a developer in the default project
	response1, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.Nil(t, err)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	response2, err := keepers.Projects.Info(ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)
}

func TestCreateProject(t *testing.T) {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	projectData := prepareProjectsData(_ctx, keepers)[1]
	plan := common.CreateMockPlan()

	subAddr := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()
	admAddr := projectData.ProjectKeys[0].Key

	err := keepers.Projects.CreateProject(ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
	ctx = sdk.UnwrapSDKContext(_ctx)

	// test invalid project name/description
	defaultProjectName := types.ADMIN_PROJECT_NAME
	longProjectName := strings.Repeat(defaultProjectName, types.MAX_PROJECT_NAME_LEN+1)
	invalidProjectName := "projectName,"

	projectDescription := "test project"
	longProjectDescription := strings.Repeat(projectDescription, types.MAX_PROJECT_DESCRIPTION_LEN+1)
	invalidProjectDescription := "projectDesc¢"

	testProjectData := projectData
	testProjectData.ProjectKeys = []types.ProjectKey{}

	nameAndDescriptionTests := []struct {
		name               string
		projectName        string
		projectDescription string
	}{
		{"bad projectName (duplicate)", projectData.Name, projectDescription},
		{"bad projectName (too long)", longProjectName, projectDescription},
		{"bad projectName (contains comma)", invalidProjectName, projectDescription},
		{"bad projectName (empty)", "", projectDescription},
		{"bad projectDescription (too long)", "test1", longProjectDescription},
		{"bad projectDescription (non ascii)", "test2", invalidProjectDescription},
	}

	for _, tt := range nameAndDescriptionTests {
		t.Run(tt.name, func(t *testing.T) {
			testProjectData.Name = tt.projectName
			testProjectData.Description = tt.projectDescription

			err = keepers.Projects.CreateProject(ctx, subAddr, testProjectData, plan)
			require.NotNil(t, err)
		})
	}

	// continue testing traits that are not related to the project's name/description
	// try creating a project with invalid project keys
	invalidKeysProjectData := projectData
	invalidKeysProjectData.Name = "nonDuplicateProjectName"
	invalidKeysProjectData.ProjectKeys = []types.ProjectKey{
		{
			Key:   subAddr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		},
		{
			Key:   admAddr,
			Types: []types.ProjectKey_KEY_TYPE{4},
		},
	}

	// should fail because there's an invalid key
	_, err = servers.SubscriptionServer.AddProject(_ctx, &subscriptiontypes.MsgAddProject{
		Creator:     subAddr,
		ProjectData: invalidKeysProjectData,
	})
	require.NotNil(t, err)

	// get project by developer - subscription key is not a developer, should fail (if it succeeds, it means that the valid project key
	// from invalidKeysProjectData was registered, which is not desired!)
	_, err = keepers.Projects.Developer(_ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.NotNil(t, err)

	response1, err := keepers.Projects.Developer(_ctx, &types.QueryDeveloperRequest{Developer: admAddr})
	require.Nil(t, err)

	response2, err := keepers.Projects.Info(_ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)

	_, err = keepers.Projects.GetProjectForBlock(ctx, response1.Project.Index, 0)
	require.Nil(t, err)

	// there should be one project key
	require.Equal(t, 1, len(response2.Project.ProjectKeys))

	// the project key is the admin key
	require.Equal(t, response2.Project.ProjectKeys[0].Key, admAddr)

	// the admin is both an admin and a developer
	require.Equal(t, 2, len(response2.Project.ProjectKeys[0].Types))
}

func TestAddKeys(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)
	_ctx := sdk.UnwrapSDKContext(ctx)
	projectData := prepareProjectsData(ctx, keepers)[2]
	plan := common.CreateMockPlan()

	subAddr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	admAddr := projectData.ProjectKeys[0].Key
	dev1Addr := projectData.ProjectKeys[1].Key
	dev2Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	dev3Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()

	err := keepers.Projects.CreateProject(_ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	projectRes, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: dev1Addr})
	require.Nil(t, err)

	project := projectRes.Project
	pk := types.ProjectKey{Key: dev1Addr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	// try adding myself as admin, should fail
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: dev1Addr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin key adding an invalid key
	pk = types.ProjectKey{Key: dev2Addr, Types: []types.ProjectKey_KEY_TYPE{4}}
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: admAddr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin key adding a developer
	pk = types.ProjectKey{Key: dev2Addr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: admAddr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// developer tries to add the second developer as admin
	pk = types.ProjectKey{Key: dev2Addr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: dev1Addr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin adding admin
	pk = types.ProjectKey{Key: dev1Addr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: admAddr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// new admin adding another developer
	pk = types.ProjectKey{Key: dev3Addr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddKeys(ctx, &types.MsgAddKeys{Creator: dev1Addr, Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// fetch project with new developer
	_, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: dev3Addr})
	require.Nil(t, err)
}

func TestAddAdminInTwoProjects(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	_ctx := sdk.UnwrapSDKContext(ctx)
	projectData := prepareProjectsData(ctx, keepers)[0]
	plan := common.CreateMockPlan()

	subAddr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	admAddr := projectData.ProjectKeys[0].Key

	err := keepers.Projects.CreateAdminProject(_ctx, subAddr, plan)
	require.Nil(t, err)

	// this is not supposed to fail because you can use the same admin key for two different projects
	// creating a regular project (not admin project) so subAccount won't be a developer there
	err = keepers.Projects.CreateProject(_ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	_, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: admAddr})
	require.NotNil(t, err)

	response, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAddr})
	require.Nil(t, err)
	require.Equal(t, response.Project.Index, types.ProjectIndex(subAddr, types.ADMIN_PROJECT_NAME))
}

func TestSetPolicy(t *testing.T) {
	SetPolicyTest(t, true)
}

func TestSetSubscriptionPolicy(t *testing.T) {
	SetPolicyTest(t, false)
}

func SetPolicyTest(t *testing.T, testAdminPolicy bool) {
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	projectData := prepareProjectsData(_ctx, keepers)[0]
	plan := common.CreateMockPlan()

	subAddr := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()
	admAddr := projectData.ProjectKeys[0].Key
	devAddr := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()

	projectID := types.ProjectIndex(subAddr, projectData.Name)

	err := keepers.Projects.CreateProject(ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	pk := types.ProjectKey{Key: devAddr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	err = keepers.Projects.AddKeysToProject(ctx, projectID, admAddr, []types.ProjectKey{pk})
	require.Nil(t, err)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(ctx, spec)

	templates := []struct {
		name                         string
		creator                      string
		projectID                    string
		geolocation                  uint64
		chainPolicies                []types.ChainPolicy
		totalCuLimit                 uint64
		epochCuLimit                 uint64
		maxProvidersToPair           uint64
		setAdminPolicySuccess        bool
		setSubscriptionPolicySuccess bool
	}{
		{
			"valid policy (admin account)", admAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false,
		},

		{
			"valid policy (subscription account)", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true,
		},

		{
			"bad creator (developer account -- not admin)", devAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			"bad projectID (doesn't exist)", devAddr, "fakeProjectId", uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			"invalid geolocation (0)", devAddr, projectID, uint64(0),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, false, false,
		},

		{
			// note: currently, we don't verify the chain policies
			"bad chainID (doesn't exist)", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: "LOL", Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true,
		},

		{
			// note: currently, we don't verify the chain policies
			"bad API (doesn't exist)", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{"lol"}}},
			100, 10, 3, true, true,
		},
		{
			// note: currently, we don't verify the chain policies
			"chainID and API not supported (exist in Lava's specs)", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: "ETH1", Apis: []string{"eth_accounts"}}},
			100, 10, 3, true, true,
		},
		{
			"epoch CU larger than total CU", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			10, 100, 3, false, false,
		},
		{
			"bad maxProvidersToPair", subAddr, projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 1, false, false,
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			newPolicy := types.Policy{
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

				_, err := servers.ProjectServer.SetPolicy(_ctx, &SetPolicyMessage)
				if tt.setAdminPolicySuccess {
					require.Nil(t, err)
					_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
					ctx = sdk.UnwrapSDKContext(_ctx)

					proj, err := keepers.Projects.GetProjectForBlock(ctx, tt.projectID, uint64(ctx.BlockHeight()))
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

				_, err := servers.ProjectServer.SetSubscriptionPolicy(_ctx, &setSubscriptionPolicyMessage)
				if tt.setSubscriptionPolicySuccess {
					require.Nil(t, err)
					_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
					ctx = sdk.UnwrapSDKContext(_ctx)

					proj, err := keepers.Projects.GetProjectForBlock(ctx, tt.projectID, uint64(ctx.BlockHeight()))
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
	servers, keepers, _ctx := testkeeper.InitAllKeepers(t)

	projectData := prepareProjectsData(_ctx, keepers)[0]
	plan := common.CreateMockPlan()

	subAddr := projectData.ProjectKeys[0].Key
	devAddr := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()

	_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
	ctx := sdk.UnwrapSDKContext(_ctx)
	block1 := uint64(ctx.BlockHeight())

	err := keepers.Projects.CreateProject(ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
	ctx = sdk.UnwrapSDKContext(_ctx)
	block2 := uint64(ctx.BlockHeight())

	projectID := types.ProjectIndex(subAddr, projectData.Name)
	project, err := keepers.Projects.GetProjectForBlock(ctx, projectID, block2)
	require.Nil(t, err)

	// add developer key (created fixation)
	pk := types.ProjectKey{Key: devAddr, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddKeys(_ctx, &types.MsgAddKeys{
		Creator:     subAddr,
		Project:     project.Index,
		ProjectKeys: []types.ProjectKey{pk},
	})
	require.Nil(t, err)

	_ctx = testkeeper.AdvanceEpoch(_ctx, keepers)
	ctx = sdk.UnwrapSDKContext(_ctx)
	block3 := uint64(ctx.BlockHeight())

	keepers.Projects.SnapshotSubscriptionProjects(ctx, subAddr)

	// try to charge CUs: should update oldest and second-oldest entries, but not the latest
	// (because the latter is in a new snapshot)

	err = keepers.Projects.ChargeComputeUnitsToProject(ctx, project, block1, 1000)
	require.Nil(t, err)

	proj, err := keepers.Projects.GetProjectForBlock(ctx, project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = keepers.Projects.GetProjectForBlock(ctx, project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = keepers.Projects.GetProjectForBlock(ctx, project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)

	keepers.Projects.ChargeComputeUnitsToProject(ctx, project, block2, 1000)

	proj, err = keepers.Projects.GetProjectForBlock(ctx, project.Index, block1)
	require.Nil(t, err)
	require.Equal(t, uint64(1000), proj.UsedCu)
	proj, err = keepers.Projects.GetProjectForBlock(ctx, project.Index, block2)
	require.Nil(t, err)
	require.Equal(t, uint64(2000), proj.UsedCu)
	proj, err = keepers.Projects.GetProjectForBlock(ctx, project.Index, block3)
	require.Nil(t, err)
	require.Equal(t, uint64(0), proj.UsedCu)
}

func TestAddDevKeyToSameProjectDifferentBlocks(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	_ctx := sdk.UnwrapSDKContext(ctx)
	projectName := "mockname1"
	subAddr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	dev1Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	dev2Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	projectID := types.ProjectIndex(subAddr, projectName)
	plan := common.CreateMockPlan()

	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   subAddr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}},
		Policy: &plan.PlanPolicy,
	}
	err := keepers.Projects.CreateProject(_ctx, subAddr, projectData, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	_ctx = sdk.UnwrapSDKContext(ctx)

	err = keepers.Projects.AddKeysToProject(_ctx, projectID, subAddr,
		[]types.ProjectKey{{
			Key:   dev1Addr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}})
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	_ctx = sdk.UnwrapSDKContext(ctx)

	err = keepers.Projects.AddKeysToProject(_ctx, projectID, subAddr,
		[]types.ProjectKey{{
			Key:   dev2Addr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}})
	require.Nil(t, err)

	proj, err := keepers.Projects.GetProjectForDeveloper(_ctx, subAddr,
		uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, 3, len(proj.ProjectKeys))
}

func TestAddDevKeyToDifferentProjectsInSameBlock(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	_ctx := sdk.UnwrapSDKContext(ctx)
	plan := common.CreateMockPlan()

	sub1Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	sub2Addr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()
	devAddr := common.CreateNewAccount(ctx, *keepers, 10000).Addr.String()

	projectName1 := "mockname1"
	projectName2 := "mockname2"

	projectID1 := types.ProjectIndex(sub1Addr, projectName1)
	projectID2 := types.ProjectIndex(sub2Addr, projectName2)

	projectData1 := types.ProjectData{
		Name:        projectName1,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   sub1Addr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}},
		Policy: &plan.PlanPolicy,
	}
	err := keepers.Projects.CreateProject(_ctx, sub1Addr, projectData1, plan)
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        projectName2,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   sub2Addr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}},
		Policy: &plan.PlanPolicy,
	}
	err = keepers.Projects.CreateProject(_ctx, sub2Addr, projectData2, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	_ctx = sdk.UnwrapSDKContext(ctx)

	err = keepers.Projects.AddKeysToProject(_ctx, projectID1, sub1Addr,
		[]types.ProjectKey{{
			Key:   devAddr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}})
	require.Nil(t, err)

	err = keepers.Projects.AddKeysToProject(_ctx, projectID2, sub2Addr,
		[]types.ProjectKey{{
			Key:   devAddr,
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
		}})
	require.NotNil(t, err) // should fail since this developer was already added to the first project

	proj1, err := keepers.Projects.GetProjectForDeveloper(_ctx, sub1Addr,
		uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	proj2, err := keepers.Projects.GetProjectForDeveloper(_ctx, sub2Addr,
		uint64(_ctx.BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, 2, len(proj1.ProjectKeys))
	require.Equal(t, 1, len(proj2.ProjectKeys))
}
