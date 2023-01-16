package allowList

type AllowList map[string]bool

// NewErrorAllowList creates a new AllowList and
// pre-populate it with specified errors
func NewErrorAllowList(errors []string) *AllowList {
	al := AllowList{}
	for _, err := range errors {
		al[err] = false
	}
	return &al
}

// SetError sets the value for the specified error name in the AllowList map to true
func (al *AllowList) SetError(name string) {
	// Check if the pointer is nil.
	if al == nil {
		// If the pointer is nil, return without modifying the map.
		return
	}

	// Check if the error name exists in the map
	if _, ok := (*al)[name]; ok {
		// If the error name exists, set its value to true
		(*al)[name] = true
	}
}

// Reset sets the value for all errors in the AllowList map to false
func (al *AllowList) Reset() {
	// Check if the pointer is nil.
	if al == nil {
		// If the pointer is nil, return without modifying the map.
		return
	}

	// Iterate over all keys and values in the map
	for k := range *al {
		// Set the value for the key to false
		(*al)[k] = false
	}
}

// IsErrorSet returns true if the value for the specified error name is set to true in the AllowList map.
// Returns false otherwise.
func (al *AllowList) IsErrorSet(name string) bool {
	// Check if the pointer is nil.
	if al == nil {
		// If the pointer is nil, return false.
		return false
	}

	// Check if the value for the specified error name is set to true.
	if val, ok := (*al)[name]; ok && val {
		return true
	}
	return false
}
