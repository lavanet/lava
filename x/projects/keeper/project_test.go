package keeper_test

import (
	"math"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/projects/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

func TestCreateDefaultProject(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)

	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	plan := common.CreateMockPlan()
	err := keepers.Projects.CreateAdminProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), plan, "")
	require.Nil(t, err)

	// subscription key is a developer in the default project
	response1, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAccount.Addr.String()})
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	response2, err := keepers.Projects.Info(ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)
}

func TestCreateProject(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	projectName := "mockName"

	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	plan := common.CreateMockPlan()

	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     false,
		ProjectKeys: []types.ProjectKey{{
			Key:   adminAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN, types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}},
		Policy: nil,
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	// test invalid project name/description
	defaultProjectName := projectstypes.ADMIN_PROJECT_NAME
	longProjectName := strings.Repeat(defaultProjectName, projectstypes.MAX_PROJECT_NAME_LEN)
	projectNameWithComma := "projectName,"
	nonAsciiProjectName := "projectName¢"

	projectDescription := "test project"
	longProjectDescription := strings.Repeat(projectDescription, projectstypes.MAX_PROJECT_DESCRIPTION_LEN)
	nonAsciiProjectDescription := "projectDesc¢"

	nameDescriptionTestProjectData := projectData
	nameDescriptionTestProjectData.ProjectKeys = []projectstypes.ProjectKey{}

	nameAndDescriptionTests := []struct {
		name               string
		projectName        string
		projectDescription string
	}{
		{"bad projectName (duplicate)", projectName, projectDescription},
		{"bad projectName (too long)", longProjectName, projectDescription},
		{"bad projectName (contains comma)", projectNameWithComma, projectDescription},
		{"bad projectName (non ascii)", nonAsciiProjectName, projectDescription},
		{"bad projectName (empty)", "", projectDescription},
		{"bad projectDescription (too long)", "test1", longProjectDescription},
		{"bad projectDescription (non ascii)", "test2", nonAsciiProjectDescription},
	}

	for _, tt := range nameAndDescriptionTests {
		t.Run(tt.name, func(t *testing.T) {
			nameDescriptionTestProjectData.Name = tt.projectName
			nameDescriptionTestProjectData.Description = tt.projectDescription

			err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), nameDescriptionTestProjectData, plan)
			require.NotNil(t, err)
		})
	}

	// continue testing traits that are not related to the project's name/description
	// try creating a project with invalid project keys
	invalidKeysProjectData := projectData
	invalidKeysProjectData.Name = "nonDuplicateProjectName"
	invalidKeysProjectData.ProjectKeys = []projectstypes.ProjectKey{
		{
			Key:   subAccount.Addr.String(),
			Types: []projectstypes.ProjectKey_KEY_TYPE{projectstypes.ProjectKey_DEVELOPER},
			Vrfpk: "",
		},
		{
			Key:   adminAcc.Addr.String(),
			Types: []projectstypes.ProjectKey_KEY_TYPE{4},
			Vrfpk: "",
		},
	}

	// should fail because there's an invalid key
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), invalidKeysProjectData, plan)
	require.NotNil(t, err)

	// get project by developer - subscription key is not a developer, should fail (if it succeeds, it means that the valid project key
	// from invalidKeysProjectData was registered, which is not desired!)
	response1, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAccount.Addr.String()})
	require.NotNil(t, err)

	response1, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.Nil(t, err)

	response2, err := keepers.Projects.Info(ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)

	proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), response1.Project.Index, 0)
	require.Nil(t, err)
	strings.Split(proj.Index, "")

	// there should be one project key
	require.Equal(t, 1, len(response2.Project.ProjectKeys))

	// the project key is the admin key
	require.Equal(t, response2.Project.ProjectKeys[0].Key, adminAcc.Addr.String())

	// the admin is both an admin and a developer
	require.Equal(t, 2, len(response2.Project.ProjectKeys[0].Types))
}

func TestAddKeys(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	projectName := "mockname"
	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	developerAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	developerAcc2 := common.CreateNewAccount(ctx, *keepers, 10000)
	plan := common.CreateMockPlan()

	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     false,
		ProjectKeys: []types.ProjectKey{
			{
				Key:   adminAcc.Addr.String(),
				Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN},
				Vrfpk: "",
			},
			{
				Key:   developerAcc.Addr.String(),
				Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
				Vrfpk: "",
			}},
		Policy: nil,
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	projectRes, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: developerAcc.Addr.String()})
	require.Nil(t, err)

	project := projectRes.Project
	pk := types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	// try adding myself as admin, should fail
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin key adding an invalid key
	pk = types.ProjectKey{Key: developerAcc2.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{4}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: adminAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin key adding a developer
	pk = types.ProjectKey{Key: developerAcc2.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: adminAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// developer tries to add the second developer as admin
	pk = types.ProjectKey{Key: developerAcc2.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin adding admin
	pk = types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: adminAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// new admin adding another developer
	developerAcc3 := common.CreateNewAccount(ctx, *keepers, 10000)
	pk = types.ProjectKey{Key: developerAcc3.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// fetch project with new developer
	projectRes, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: developerAcc3.Addr.String()})
	require.Nil(t, err)
}

func TestAddAdminInTwoProjects(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	// he should be a developer only in the first project
	projectName := "mockname"

	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	plan := common.CreateMockPlan()
	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     false,
		ProjectKeys: []types.ProjectKey{{
			Key:   adminAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN},
			Vrfpk: "",
		}},
		Policy: &types.Policy{GeolocationProfile: math.MaxUint64},
	}
	err := keepers.Projects.CreateAdminProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), plan, "")
	require.Nil(t, err)

	// this is not supposed to fail because you can use the same admin key for two different projects
	// creating a regular project (not admin project) so subAccount won't be a developer there
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	response, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.NotNil(t, err)
	response, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAccount.Addr.String()})
	require.Nil(t, err)
	require.Equal(t, response.Project.Index, types.ProjectIndex(subAccount.Addr.String(), types.ADMIN_PROJECT_NAME))
}

func TestSetAdminPolicy(t *testing.T) {
	SetPolicyTest(t, true)
}

func TestSetSubscriptionPolicy(t *testing.T) {
	SetPolicyTest(t, false)
}

func SetPolicyTest(t *testing.T, testAdminPolicy bool) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	projectName := "mockname1"
	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	developerAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	projectID := types.ProjectIndex(subAccount.Addr.String(), projectName)
	plan := common.CreateMockPlan()

	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   adminAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN},
			Vrfpk: "",
		}},
		Policy: &types.Policy{GeolocationProfile: math.MaxUint64},
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	err = keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID, adminAcc.Addr.String(),
		[]types.ProjectKey{{
			Key:   developerAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})
	require.Nil(t, err)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	templates := []struct {
		name                 string
		creator              string
		projectID            string
		geolocation          uint64
		chainPolicies        []types.ChainPolicy
		totalCuLimit         uint64
		epochCuLimit         uint64
		maxProvidersToPair   uint64
		validateBasicSuccess bool
		setPolicySuccess     bool
	}{
		{"valid policy (admin account)", adminAcc.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true},

		{"valid policy (subscription account)", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true},

		{"bad creator (developer account -- not admin)", developerAcc.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false},

		{"bad projectID (doesn't exist)", developerAcc.Addr.String(), "fakeProjectId", uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false},

		{"invalid geolocation (0)", developerAcc.Addr.String(), "fakeProjectId", uint64(0),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false},

		{"bad chainID (doesn't exist)", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: "LOL", Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true}, // note: currently, we don't verify the chain policies

		{"bad API (doesn't exist)", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{"lol"}}},
			100, 10, 3, true, true}, // note: currently, we don't verify the chain policies
		{"chainID and API not supported (exist in Lava's specs)", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: "ETH1", Apis: []string{"eth_accounts"}}},
			100, 10, 3, true, true}, // note: currently, we don't verify the chain policies
		{"epoch CU larger than total CU", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			10, 100, 3, false, false},
		{"bad maxProvidersToPair", subAccount.Addr.String(), projectID, uint64(1),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 1, false, false},
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
				setAdminPolicyMessage := types.MsgSetAdminPolicy{
					Creator: tt.creator,
					Policy:  newPolicy,
					Project: tt.projectID,
				}

				err = setAdminPolicyMessage.ValidateBasic()
				if tt.validateBasicSuccess {
					require.Nil(t, err)
				} else {
					require.NotNil(t, err)
					return
				}

				_, err := servers.ProjectServer.SetAdminPolicy(ctx, &setAdminPolicyMessage)
				if tt.setPolicySuccess {
					require.Nil(t, err)
					ctx = testkeeper.AdvanceEpoch(ctx, keepers)

					proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), tt.projectID, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
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
				if tt.validateBasicSuccess {
					require.Nil(t, err)
				} else {
					require.NotNil(t, err)
					return
				}

				_, err := servers.ProjectServer.SetSubscriptionPolicy(ctx, &setSubscriptionPolicyMessage)
				if tt.creator == subAccount.Addr.String() {
					// only the subscription consumer should be able to set subscription policy
					require.Nil(t, err)

					if tt.setPolicySuccess {
						require.Nil(t, err)
						ctx = testkeeper.AdvanceEpoch(ctx, keepers)

						proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), tt.projectID, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
						require.Nil(t, err)

						require.Equal(t, newPolicy, *proj.SubscriptionPolicy)
					} else {
						require.NotNil(t, err)
					}
				} else {
					require.NotNil(t, err)
				}
			}
		})
	}
}

func TestAddDevKeyToSameProjectDifferentBlocks(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)

	projectName := "mockname1"
	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	devAcc1 := common.CreateNewAccount(ctx, *keepers, 10000)
	devAcc2 := common.CreateNewAccount(ctx, *keepers, 10000)
	projectID := types.ProjectIndex(subAccount.Addr.String(), projectName)
	plan := common.CreateMockPlan()

	projectData := types.ProjectData{
		Name:        projectName,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   subAccount.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}},
		Policy: &plan.PlanPolicy,
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	err = keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID, subAccount.Addr.String(),
		[]types.ProjectKey{{
			Key:   devAcc1.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	err = keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID, subAccount.Addr.String(),
		[]types.ProjectKey{{
			Key:   devAcc2.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})
	require.Nil(t, err)

	proj, _, err := keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(),
		uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, 3, len(proj.ProjectKeys))
}

func TestAddDevKeyToDifferentProjectsInSameBlock(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	plan := common.CreateMockPlan()

	subAccount1 := common.CreateNewAccount(ctx, *keepers, 10000)
	subAccount2 := common.CreateNewAccount(ctx, *keepers, 10000)
	devAcc := common.CreateNewAccount(ctx, *keepers, 10000)

	projectName1 := "mockname1"
	projectName2 := "mockname2"

	projectID1 := types.ProjectIndex(subAccount1.Addr.String(), projectName1)
	projectID2 := types.ProjectIndex(subAccount2.Addr.String(), projectName2)

	projectData1 := types.ProjectData{
		Name:        projectName1,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   subAccount1.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}},
		Policy: &plan.PlanPolicy,
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount1.Addr.String(), projectData1, plan)
	require.Nil(t, err)

	projectData2 := types.ProjectData{
		Name:        projectName2,
		Description: "",
		Enabled:     true,
		ProjectKeys: []types.ProjectKey{{
			Key:   subAccount2.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}},
		Policy: &plan.PlanPolicy,
	}
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount2.Addr.String(), projectData2, plan)
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	err = keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID1, subAccount1.Addr.String(),
		[]types.ProjectKey{{
			Key:   devAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})
	require.Nil(t, err)

	err = keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID2, subAccount2.Addr.String(),
		[]types.ProjectKey{{
			Key:   devAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})
	require.NotNil(t, err) // should fail since this developer was already added to the first project

	proj1, _, err := keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ctx), subAccount1.Addr.String(),
		uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)

	proj2, _, err := keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ctx), subAccount2.Addr.String(),
		uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)

	require.Equal(t, 2, len(proj1.ProjectKeys))
	require.Equal(t, 1, len(proj2.ProjectKeys))
}
