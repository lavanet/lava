package types

import (
	"fmt"
)

const ADMIN_PROJECT_NAME = "admin"

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func CreateProject(subscriptionAddress string, projectName string) Project {
	return Project{
		Index:        ProjectIndex(subscriptionAddress, projectName),
		Subscription: subscriptionAddress,
		Description:  "",
		ProjectKeys:  []ProjectKey{},
		Policy:       Policy{},
		UsedCu:       0,
	}
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

func (project *Project) VerifyProject(chainID string) error {
	if !project.Enabled {
		return fmt.Errorf("the developers project is disabled")
	}

	if !project.Policy.ContainsChainID(chainID) {
		return fmt.Errorf("the developers project policy does not include the chain")
	}

	if project.Policy.TotalCuLimit <= project.UsedCu {
		return fmt.Errorf("the developers project policy used all the allowed cu for this project")
	}

	return nil
}
