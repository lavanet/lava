package types

const DEFAULT_PROJECT_NAME = "default"

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func CreateEmptyProject(subscriptionAddress string, projectName string) Project {
	return Project{
		Index:        ProjectIndex(subscriptionAddress, projectName),
		Subscription: subscriptionAddress,
		Description:  "Permissive default project",
		ProjectKeys:  []ProjectKey{},
		Policy:       Policy{},
		UsedCu:       0,
	}
}

func DefaultProject(subscriptionAddress string) Project {
	return CreateEmptyProject(subscriptionAddress, DEFAULT_PROJECT_NAME)
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

func (project *Project) IsKeyType(projectKey string, keyTypeToCheck ProjectKey_KEY_TYPE) bool {
	return project.GetKey(projectKey).IsKeyType(keyTypeToCheck)
}
