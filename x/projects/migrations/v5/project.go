package types

func NewProjectKey(key string, kinds uint32) ProjectKey {
	return ProjectKey{Key: key, Kinds: kinds}
}

func (projectKey ProjectKey) AddType(kind ProjectKey_Type) ProjectKey {
	projectKey.Kinds |= uint32(kind)
	return projectKey
}
