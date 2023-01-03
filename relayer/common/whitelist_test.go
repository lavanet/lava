package common

import "testing"

// TestSetError tests the SetError method of the EpochErrorWhitelist data structure
func TestSetError(t *testing.T) {
	// Create a new EpochErrorWhitelist with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	whitelist := NewEpochErrorWhitelist(errors)

	// Set the value for "error 1" to true
	whitelist.SetError("error 1")

	// Check that the value for "error 1" is set to true
	if whitelist.IsErrorSet("error 1") != true {
		t.Error("Expected error 1 value to be true, got false")
	}

	// Set the value for "error 4" to true
	whitelist.SetError("error 4")

	// Check that the value for "error 4" is not set to true
	if whitelist.IsErrorSet("error 4") != false {
		t.Error("Expected error 4 value to be false, got true")
	}
}

// TestReset tests the Reset method of the EpochErrorWhitelist data structure
func TestReset(t *testing.T) {
	// Create a new EpochErrorWhitelist with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	whitelist := NewEpochErrorWhitelist(errors)

	// Set the value for "error 1" to true
	whitelist.SetError("error 1")

	// Reset the values for all errors
	whitelist.Reset()

	// Check that the values for all errors are set to false
	for _, err := range errors {
		if whitelist.IsErrorSet(err) != false {
			t.Errorf("Expected %s value to be false, got true", err)
		}
	}
}

// TestNilPointer tests the EpochErrorWhitelist data structure and its associated methods for nil pointer handling
func TestNilPointer(t *testing.T) {
	// Define a function to handle panic situations
	defer func() {
		if r := recover(); r != nil {
			t.Error("Unexpected panic occurred")
		}
	}()

	// Create a nil pointer for the EpochErrorWhitelist data structure
	var nilWhitelist *EpochErrorWhitelist

	// Try to reset the values for all errors on null pointer
	nilWhitelist.Reset()

	// Try to set the value for "error 1" to true on null pointer
	nilWhitelist.SetError("error 1")

	// Try to check if error exists on null pointer
	if nilWhitelist.IsErrorSet("error 1") != false {
		t.Error("Expected false, got true")
	}

}
