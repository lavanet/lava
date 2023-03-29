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
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false, "", plan, math.MaxUint64, "")
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	// create another project with the same name, should fail as this is unique
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false, "", plan, math.MaxUint64, "")
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
	plan := common.CreateMockPlan()
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false, "", plan, math.MaxUint64, "")
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	projectRes, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.Nil(t, err)

	project := projectRes.Project
	pk := types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	// try adding myself as admin, should fail
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin key adding as developer
	pk = types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: adminAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// developer tries to add admin
	pk = types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.NotNil(t, err)

	// admin adding admin
	pk = types.ProjectKey{Key: developerAcc.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: adminAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// new admin adding another developer
	developerAcc2 := common.CreateNewAccount(ctx, *keepers, 10000)
	pk = types.ProjectKey{Key: developerAcc2.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}}
	_, err = servers.ProjectServer.AddProjectKeys(ctx, &types.MsgAddProjectKeys{Creator: developerAcc.Addr.String(), Project: project.Index, ProjectKeys: []types.ProjectKey{pk}})
	require.Nil(t, err)

	// fetch project with new developer
	projectRes, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: developerAcc2.Addr.String()})
	require.Nil(t, err)
}

func TestAddAdminInTwoProjects(t *testing.T) {
	_, keepers, ctx := testkeeper.InitAllKeepers(t)
	// he should be a developer only in the first project
	projectName1 := "mockname1"
	projectName2 := "mockname2"

	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	plan := common.CreateMockPlan()
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName1, adminAcc.Addr.String(), false, "", plan, math.MaxUint64, "")
	require.Nil(t, err)

	// this is supposed to fail because you can't use the same admin key for two different projects
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName2, adminAcc.Addr.String(), false, "", plan, math.MaxUint64, "")
	require.NotNil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	response, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.Nil(t, err)
	require.Equal(t, response.Project.Index, types.ProjectIndex(subAccount.Addr.String(), projectName1))
}

func TestSetPolicy(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	projectName := "mockname1"
	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	developerAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	projectID := types.ProjectIndex(subAccount.Addr.String(), projectName)

	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), true, 100, 100, 5, math.MaxUint64, "")
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
		name                           string
		creator                        string
		chainPolicies                  []types.ChainPolicy
		totalCuLimit                   uint64
		epochCuLimit                   uint64
		maxProvidersToPair             uint64
		basicValidationSuccess         bool
		chainPoliciesValidationSuccess bool
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

		{"bad chainID (doesn't exist)", adminAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: "LOL", Apis: []string{spec.Apis[0].Name}}},
			100, 10, 3, true, false},

		{"bad API (doesn't exist)", adminAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{"lol"}}},
			100, 10, 3, true, false},
		{"epoch CU larger than total CU", adminAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			10, 100, 3, false, true},
		{"bad maxProvidersToPair", adminAcc.Addr.String(),
			[]types.ChainPolicy{{ChainId: spec.Index, Apis: []string{spec.Apis[0].Name}}},
			100, 10, 0, false, true},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			newPolicy := types.Policy{
				ChainPolicies:      tt.chainPolicies,
				GeolocationProfile: 1,
				TotalCuLimit:       tt.totalCuLimit,
				EpochCuLimit:       tt.epochCuLimit,
				MaxProvidersToPair: tt.maxProvidersToPair,
			}

			setPolicyProjectMessage := types.MsgSetProjectPolicy{
				Creator: tt.creator,
				Policy:  &newPolicy,
				Project: projectID,
			}

			err = setPolicyProjectMessage.ValidateBasic()
			if tt.basicValidationSuccess {
				require.Nil(t, err)
			} else {
				require.NotNil(t, err)
			}

			_, err := servers.ProjectServer.SetProjectPolicy(ctx, &setPolicyProjectMessage)

			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			if tt.chainPoliciesValidationSuccess {
				require.Nil(t, err)

				proj, err := keepers.Projects.GetProjectForBlock(sdk.UnwrapSDKContext(ctx), projectID, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
				require.Nil(t, err)
				require.Equal(t, tt.chainPolicies, proj.Policy.ChainPolicies)
				require.Equal(t, uint64(1), proj.Policy.GeolocationProfile)
				require.Equal(t, tt.totalCuLimit, proj.Policy.TotalCuLimit)
				require.Equal(t, tt.epochCuLimit, proj.Policy.EpochCuLimit)
				require.Equal(t, tt.maxProvidersToPair, proj.Policy.MaxProvidersToPair)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
