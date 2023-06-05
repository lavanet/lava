package types

const (
	MAX_PROJECT_NAME_LEN        = 50
	MAX_PROJECT_DESCRIPTION_LEN = 150
)

// set policy enum
type SetPolicyEnum int

const (
	SET_ADMIN_POLICY        SetPolicyEnum = 1
	SET_SUBSCRIPTION_POLICY SetPolicyEnum = 2
)
