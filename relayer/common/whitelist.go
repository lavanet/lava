package common

type EpochErrorWhitelist map[string]bool

// NewEpochErrorWhitelist creates a new epochErrorWhitelist and
// pre-populate it with specified errors
func NewEpochErrorWhitelist(errors []string) *EpochErrorWhitelist {
	wl := EpochErrorWhitelist{}
	for _, err := range errors {
		wl[err] = false
	}
	return &wl
}

// SetError sets the value for the specified error name in the EpochErrorWhitelist map to true
func (wl *EpochErrorWhitelist) SetError(name string) {
	// Check if the pointer is nil.
	if wl == nil {
		// If the pointer is nil, return without modifying the map.
		return
	}

	// Check if the error name exists in the map
	if _, ok := (*wl)[name]; ok {
		// If the error name exists, set its value to true
		(*wl)[name] = true
	}
}

// Reset sets the value for all errors in the EpochErrorWhitelist map to false
func (wl *EpochErrorWhitelist) Reset() {
	// Check if the pointer is nil.
	if wl == nil {
		// If the pointer is nil, return without modifying the map.
		return
	}

	// Iterate over all keys and values in the map
	for k := range *wl {
		// Set the value for the key to false
		(*wl)[k] = false
	}
}

// IsErrorSet returns true if the value for the specified error name is set to true in the EpochErrorWhitelist map.
// Returns false otherwise.
func (wl *EpochErrorWhitelist) IsErrorSet(name string) bool {
	// Check if the pointer is nil.
	if wl == nil {
		// If the pointer is nil, return false.
		return false
	}

	// Check if the value for the specified error name is set to true.
	if val, ok := (*wl)[name]; ok && val {
		return true
	}
	return false
}
