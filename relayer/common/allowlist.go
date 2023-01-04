package common

type EpochErrorAllowlist map[string]bool

// NewEpochErrorAllowlist creates a new EpochErrorAllowlist and
// pre-populate it with specified errors
func NewEpochErrorAllowlist(errors []string) *EpochErrorAllowlist {
	al := EpochErrorAllowlist{}
	for _, err := range errors {
		al[err] = false
	}
	return &al
}

// SetError sets the value for the specified error name in the EpochErrorAllowlist map to true
func (al *EpochErrorAllowlist) SetError(name string) {
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

// Reset sets the value for all errors in the EpochErrorAllowlist map to false
func (al *EpochErrorAllowlist) Reset() {
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

// IsErrorSet returns true if the value for the specified error name is set to true in the EpochErrorAllowlist map.
// Returns false otherwise.
func (al *EpochErrorAllowlist) IsErrorSet(name string) bool {
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
