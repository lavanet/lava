package types

const DEFAULT_PROJECT_NAME = "default"

func ProjectIndex(subscriptionAddress string, projectName string) string {
	return subscriptionAddress + "-" + projectName
}

func CreateEmptyProject(subscriptionAddress string, projectName string) Project {
	return Project{
		Index:        ProjectIndex(subscriptionAddress, projectName),
		Subscription: subscriptionAddress,
		Description:  "Default project that has all the available resources of the subscription",
		ProjectKeys:  []*ProjectKey{},
		Policy:       &Policy{},
		UsedCu:       0,
	}
}

func DefualtProject(subscriptionAddress string) Project {
	return CreateEmptyProject(subscriptionAddress, DEFAULT_PROJECT_NAME)
}
