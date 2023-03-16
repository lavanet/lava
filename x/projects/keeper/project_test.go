package keeper_test

import (
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
	err := keepers.Projects.CreateDefaultProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String())
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
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	// create another project with the same name, should fail as this is unique
	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false)
	require.NotNil(t, err)

	// subscription key is not a developer
	response1, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: subAccount.Addr.String()})
	require.NotNil(t, err)

	response1, err = keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.Nil(t, err)

	response2, err := keepers.Projects.Info(ctx, &types.QueryInfoRequest{Project: response1.Project.Index})
	require.Nil(t, err)

	require.Equal(t, response2.Project, response1.Project)

	require.Equal(t, len(response2.Project.ProjectKeys), 1)
	require.Equal(t, response2.Project.ProjectKeys[0].Key, adminAcc.Addr.String())
	require.Equal(t, len(response2.Project.ProjectKeys[0].Types), 1)
	require.Equal(t, response2.Project.ProjectKeys[0].Types[0], types.ProjectKey_ADMIN)
}

func TestAddKeys(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	projectName := "mockname"
	subAccount := common.CreateNewAccount(ctx, *keepers, 10000)
	adminAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	developerAcc := common.CreateNewAccount(ctx, *keepers, 10000)
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName, adminAcc.Addr.String(), false)
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
	pk = types.ProjectKey{Key: developerAcc2.Addr.String(), Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}}
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
	err := keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName1, adminAcc.Addr.String(), false)
	require.Nil(t, err)

	err = keepers.Projects.CreateProject(sdk.UnwrapSDKContext(ctx), subAccount.Addr.String(), projectName2, adminAcc.Addr.String(), false)
	require.Nil(t, err)

	testkeeper.AdvanceEpoch(ctx, keepers)

	response, err := keepers.Projects.Developer(ctx, &types.QueryDeveloperRequest{Developer: adminAcc.Addr.String()})
	require.Nil(t, err)
	require.Equal(t, response.Project.Index, types.ProjectIndex(subAccount.Addr.String(), projectName1))
}
