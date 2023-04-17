package types

import (
	"fmt"
	"strings"

	commontypes "github.com/lavanet/lava/common/types"
)

const (
	ADMIN_PROJECT_NAME        = "admin"
	ADMIN_PROJECT_DESCRIPTION = "default admin project"
)

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func CreateProject(subscriptionAddress string, projectName string, description string, enable bool) (Project, error) {
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
	if !strings.Contains(name, ",") || !commontypes.IsASCII(name) ||
		len(name) > MAX_PROJECT_NAME_LEN || len(description) > MAX_PROJECT_DESCRIPTION_LEN ||
		name == "" {
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
