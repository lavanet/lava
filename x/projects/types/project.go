package types

import (
	"fmt"

	commontypes "github.com/lavanet/lava/v2/utils/common/types"
)

const (
	ADMIN_PROJECT_NAME = "admin"
)

func ProjectIndex(subscriptionAddress, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func NewProject(subscriptionAddress, projectName string, enable bool) (Project, error) {
	if !ValidateProjectName(projectName) {
		return Project{}, fmt.Errorf("invalid project name: %s", projectName)
	}

	return Project{
		Index:              ProjectIndex(subscriptionAddress, projectName),
		Subscription:       subscriptionAddress,
		ProjectKeys:        []ProjectKey{},
		AdminPolicy:        nil,
		SubscriptionPolicy: nil,
		UsedCu:             0,
		Enabled:            enable,
	}, nil
}

func ValidateProjectName(name string) bool {
	if !commontypes.ValidateString(name, commontypes.NAME_RESTRICTIONS, nil) ||
		len(name) > MAX_PROJECT_NAME_LEN {
		return false
	}

	return true
}

func NewProjectKey(key string) ProjectKey {
	return ProjectKey{Key: key}
}

func (projectKey ProjectKey) AddType(kind ProjectKey_Type) ProjectKey {
	projectKey.Kinds |= uint32(kind)
	return projectKey
}

func (projectKey ProjectKey) SetType(kind ProjectKey_Type) ProjectKey {
	projectKey.Kinds = uint32(kind)
	return projectKey
}

func ProjectAdminKey(key string) ProjectKey {
	return NewProjectKey(key).AddType(ProjectKey_ADMIN)
}

func ProjectDeveloperKey(key string) ProjectKey {
	return NewProjectKey(key).AddType(ProjectKey_DEVELOPER)
}

func (projectKey ProjectKey) IsType(kind ProjectKey_Type) bool {
	return projectKey.Kinds&uint32(kind) != 0x0
}

func (projectKey ProjectKey) IsTypeValid() bool {
	const keyKindsAll = (uint32(ProjectKey_ADMIN) | uint32(ProjectKey_DEVELOPER))

	return projectKey.Kinds != uint32(ProjectKey_NONE) &&
		(projectKey.Kinds & ^keyKindsAll) == 0x0
}

func (project *Project) GetKey(key string) ProjectKey {
	for _, projectKey := range project.ProjectKeys {
		if projectKey.Key == key {
			return projectKey
		}
	}
	return ProjectKey{}
}

func (project *Project) AppendKey(key ProjectKey) bool {
	for i, projectKey := range project.ProjectKeys {
		if projectKey.Key == key.Key {
			project.ProjectKeys[i].Kinds |= key.Kinds
			return true
		}
	}
	project.ProjectKeys = append(project.ProjectKeys, key)
	return false
}

func (project *Project) DeleteKey(key ProjectKey) bool {
	length := len(project.ProjectKeys)
	for i, projectKey := range project.ProjectKeys {
		if projectKey.Key == key.Key {
			project.ProjectKeys[i].Kinds &= ^key.Kinds
			if project.ProjectKeys[i].Kinds == uint32(ProjectKey_NONE) {
				if i < length-1 {
					project.ProjectKeys[i] = project.ProjectKeys[length-1]
				}
				project.ProjectKeys = project.ProjectKeys[0 : length-1]
			}
			return true
		}
	}
	return false
}

func (project *Project) IsAdminKey(key string) bool {
	return project.Subscription == key || project.GetKey(key).IsType(ProjectKey_ADMIN)
}
