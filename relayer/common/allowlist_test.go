package common

import "testing"

// TestSetError tests the SetError method of the EpochErrorAllowlist data structure
func TestSetError(t *testing.T) {
	// Create a new EpochErrorAllowlist with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	allowlist := NewEpochErrorAllowlist(errors)

	// Set the value for "error 1" to true
	allowlist.SetError("error 1")

	// Check that the value for "error 1" is set to true
	if allowlist.IsErrorSet("error 1") != true {
		t.Error("Expected error 1 value to be true, got false")
	}

	// Set the value for "error 4" to true
	allowlist.SetError("error 4")

	// Check that the value for "error 4" is not set to true
	if allowlist.IsErrorSet("error 4") != false {
		t.Error("Expected error 4 value to be false, got true")
	}
}

// TestReset tests the Reset method of the EpochErrorAllowlist data structure
func TestReset(t *testing.T) {
	// Create a new EpochErrorAllowlist with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	allowlist := NewEpochErrorAllowlist(errors)

	// Set the value for "error 1" to true
	allowlist.SetError("error 1")

	// Reset the values for all errors
	allowlist.Reset()

	// Check that the values for all errors are set to false
	for _, err := range errors {
		if allowlist.IsErrorSet(err) != false {
			t.Errorf("Expected %s value to be false, got true", err)
		}
	}
}

// TestNilPointer tests the EpochErrorAllowlist data structure and its associated methods for nil pointer handling
func TestNilPointer(t *testing.T) {
	// Define a function to handle panic situations
	defer func() {
		if r := recover(); r != nil {
			t.Error("Unexpected panic occurred")
		}
	}()

	// Create a nil pointer for the EpochErrorAllowlist data structure
	var nilAllowlist *EpochErrorAllowlist

	// Try to reset the values for all errors on null pointer
	nilAllowlist.Reset()

	// Try to set the value for "error 1" to true on null pointer
	nilAllowlist.SetError("error 1")

	// Try to check if error exists on null pointer
	if nilAllowlist.IsErrorSet("error 1") != false {
		t.Error("Expected false, got true")
	}

}
