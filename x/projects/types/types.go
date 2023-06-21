package types

const (
	MAX_PROJECT_NAME_LEN = 50
)

// set policy enum
type SetPolicyEnum int

const (
	SET_ADMIN_POLICY        SetPolicyEnum = 1
	SET_SUBSCRIPTION_POLICY SetPolicyEnum = 2
)

const (
	AddProjectKeyEventName = "add_key_to_project_event"
	DelProjectKeyEventName = "del_key_from_project_event"
)
