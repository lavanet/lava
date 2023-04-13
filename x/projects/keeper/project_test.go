package keeper_test

import (
	"math"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/projects/types"
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
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN, types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}},
		Policy: nil,
	}
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	// create another project with the same name, should fail as this is unique
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectData, plan)
	require.NotNil(t, err)

	// subscription key is not a developer
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

	keepers.Projects.AddKeysToProject(sdk.UnwrapSDKContext(ctx), projectID, adminAcc.Addr.String(),
		[]types.ProjectKey{{
			Key:   developerAcc.Addr.String(),
			Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER},
			Vrfpk: "",
		}})

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	templates := []struct {
		name                 string
		creator              string
		chainPolicies        []types.ChainPolicy
		totalCuLimit         uint64
		epochCuLimit         uint64
		maxProvidersToPair   uint64
		validateBasicSuccess bool
		setPolicySuccess     bool
	}{
		{"valid policy (admin account)", adminAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true},

		{"valid policy (subscription account)", subAccount.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true},

		{"bad creator (developer account -- not admin)", developerAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false},

		{"bad chainID (doesn't exist)", subAccount.Addr.String(),
			[]types.ChainPolicy{{ChainId: "LOL", Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, true}, // note: currently, we don't verify the chain policies

		{"bad API (doesn't exist)", subAccount.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{"lol"}}},
			100, 10, 3, true, true}, // note: currently, we don't verify the chain policies
		{"epoch CU larger than total CU", subAccount.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			10, 100, 3, false, false},
		{"bad maxProvidersToPair", subAccount.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 1, false, false},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			newPolicy := types.Policy{
				ChainPolicies:      tt.chainPolicies,
				GeolocationProfile: uint64(1),
				TotalCuLimit:       tt.totalCuLimit,
				EpochCuLimit:       tt.epochCuLimit,
				MaxProvidersToPair: tt.maxProvidersToPair,
			}

			if testAdminPolicy {
				setAdminPolicyMessage := types.MsgSetAdminPolicy{
					Creator: tt.creator,
					Policy:  newPolicy,
					Project: projectID,
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

					proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), projectID, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
					require.Nil(t, err)

					require.Equal(t, newPolicy, proj.AdminPolicy)
				} else {
					require.NotNil(t, err)
				}

			} else {
				setSubscriptionPolicyMessage := types.MsgSetSubscriptionPolicy{
					Creator:  tt.creator,
					Policy:   newPolicy,
					Projects: []string{projectID},
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

						proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), projectID, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
						require.Nil(t, err)

						require.Equal(t, newPolicy, proj.SubscriptionPolicy)
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
