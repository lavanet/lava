package types

import (
	"fmt"
	"strings"
	"unicode"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const ADMIN_PROJECT_NAME = "admin"

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func CreateProject(subscriptionAddress string, projectName string) (Project, error) {
	if !validateProjectName(projectName) {
		return Project{}, fmt.Errorf("project name must be ASCII and cannot contain \",\". Name: %s", projectName)
	}

	return Project{
		Index:              ProjectIndex(subscriptionAddress, projectName),
		Subscription:       subscriptionAddress,
		Description:        "",
		ProjectKeys:        []ProjectKey{},
		AdminPolicy:        Policy{},
		SubscriptionPolicy: Policy{},
		UsedCu:             0,
	}, nil
}

func validateProjectName(projectName string) bool {
	if strings.Contains(projectName, ",") || !isASCII(projectName) {
		return false
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (project *Project) GetKey(projectKey string) ProjectKey {
	for _, key := range project.ProjectKeys {
		if key.Key == projectKey {
			return key
		}
	}
	return ProjectKey{}
}

func (projectKey ProjectKey) IsKeyType(keyTypeToCheck ProjectKey_KEY_TYPE) bool {
	for _, keytype := range projectKey.Types {
		if keytype == keyTypeToCheck {
			return true
		}
	}
	return false
}

func (projectKey *ProjectKey) AppendKeyType(typesToAdd []ProjectKey_KEY_TYPE) {
	for _, keytype := range typesToAdd {
		if !projectKey.IsKeyType(keytype) {
			projectKey.Types = append(projectKey.Types, keytype)
		}
	}
}

func (project *Project) AppendKey(keyToAdd ProjectKey) {
	for i := 0; i < len(project.ProjectKeys); i++ {
		if project.ProjectKeys[i].Key == keyToAdd.Key {
			project.ProjectKeys[i].AppendKeyType(keyToAdd.Types)
			return
		}
	}
	project.ProjectKeys = append(project.ProjectKeys, keyToAdd)
}

func (project *Project) HasKeyType(projectKey string, keyTypeToCheck ProjectKey_KEY_TYPE) bool {
	return project.GetKey(projectKey).IsKeyType(keyTypeToCheck)
}

func (project *Project) IsAdminKey(projectKey string) bool {
	return project.HasKeyType(projectKey, ProjectKey_ADMIN) || project.Subscription == projectKey
}

func (project *Project) VerifyProject(chainID string) error {
	if !project.Enabled {
		return fmt.Errorf("the developers project is disabled")
	}

	if !project.AdminPolicy.ContainsChainID(chainID) {
		return fmt.Errorf("the developers project policy does not include the chain")
	}

	err := project.VerifyCuUsage()
	return err
}

func (project *Project) VerifyCuUsage() error {
	// TODO: when overuse is added, change here to take that into account
	if project.AdminPolicy.TotalCuLimit <= project.UsedCu {
		return fmt.Errorf("the developers project policy used all the allowed cu for this project")
	}
	return nil
}

func ValidateBasicPolicy(policy Policy) error {
	if policy.EpochCuLimit > policy.TotalCuLimit {
		return sdkerrors.Wrapf(ErrInvalidPolicyCuFields, "invalid policy's CU fields (EpochCuLimit = %v, TotalCuLimit = %v)", policy.EpochCuLimit, policy.TotalCuLimit)
	}

	if policy.MaxProvidersToPair <= 1 {
		return sdkerrors.Wrapf(ErrInvalidPolicyMaxProvidersToPair, "invalid policy's MaxProvidersToPair fields (MaxProvidersToPair = %v)", policy.MaxProvidersToPair)
	}

	return nil
}
