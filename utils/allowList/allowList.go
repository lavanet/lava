package allowList

type AllowList map[uint32]bool

// NewErrorAllowList creates a new AllowList and
// pre-populate it with specified errors
// not thread safe
func NewErrorAllowList(errors []uint32) *AllowList {
	al := AllowList{}
	for _, err := range errors {
		al[err] = false
	}
	return &al
}

// SetError sets the value for the specified error name in the AllowList map to true
func (al *AllowList) SetError(code uint32) {
	// Check if the pointer is nil.
	if al == nil {
		// If the pointer is nil, return without modifying the map.
		return
	}

	// Check if the error name exists in the map
	if _, ok := (*al)[code]; ok {
		// If the error name exists, set its value to true
		(*al)[code] = true
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
func (al *AllowList) IsErrorSet(code uint32) bool {
	// Check if the pointer is nil.
	if al == nil {
		// If the pointer is nil, return false.
		return false
	}

	return (*al)[code]
}
