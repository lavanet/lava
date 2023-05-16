package types

import (
	"fmt"

	commontypes "github.com/lavanet/lava/common/types"
)

const (
	ADMIN_PROJECT_NAME        = "admin"
	ADMIN_PROJECT_DESCRIPTION = "default admin project"
)

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func NewProject(subscriptionAddress string, projectName string, description string, enable bool) (Project, error) {
	if !ValidateProjectNameAndDescription(projectName, description) {
		return Project{}, fmt.Errorf("project name must be ASCII, cannot contain \",\" and its length must be less than %d."+
			" Name: %s. The project's description must also be ASCII and its length must be less than %d",
			MAX_PROJECT_NAME_LEN, projectName, MAX_PROJECT_DESCRIPTION_LEN)
	}

	return Project{
		Index:              ProjectIndex(subscriptionAddress, projectName),
		Subscription:       subscriptionAddress,
		Description:        description,
		ProjectKeys:        []ProjectKey{},
		AdminPolicy:        nil,
		SubscriptionPolicy: nil,
		UsedCu:             0,
		Enabled:            enable,
	}, nil
}

func ValidateProjectNameAndDescription(name string, description string) bool {
	if !commontypes.ValidateString(name, commontypes.NAME_RESTRICTIONS, nil) ||
		len(name) > MAX_PROJECT_NAME_LEN || len(description) > MAX_PROJECT_DESCRIPTION_LEN ||
		!commontypes.ValidateString(description, commontypes.DESCRIPTION_RESTRICTIONS, nil) {
		return false
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

func (projectKey ProjectKey) IsKeyType(keyTypeToCheck ProjectKey_KeyType) bool {
	for _, keyType := range projectKey.Types {
		if keyType.KeyTypes == keyTypeToCheck.KeyTypes {
			return true
		}
	}
	return false
}

func (projectKey *ProjectKey) AppendKeyType(typesToAdd []ProjectKey_KeyType) {
	for _, keyType := range typesToAdd {
		if !projectKey.IsKeyType(keyType) {
			projectKey.Types = append(projectKey.Types, keyType)
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

func (project *Project) HasKeyType(projectKey string, keyTypeToCheck ProjectKey_KeyType) bool {
	return project.GetKey(projectKey).IsKeyType(keyTypeToCheck)
}

func (project *Project) IsAdminKey(projectKey string) bool {
	return project.HasKeyType(projectKey, ProjectKey_KeyType{KeyTypes: ProjectKey_ADMIN}) || project.Subscription == projectKey
}
