package allowList

import "testing"

// TestSetError tests the SetError method of the AllowList data structure
func TestSetError(t *testing.T) {
	// Create a new AllowList with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	allowList := NewErrorAllowList(errors)

	// Set the value for "error 1" to true
	allowList.SetError("error 1")

	// Check that the value for "error 1" is set to true
	if allowList.IsErrorSet("error 1") != true {
		t.Error("Expected error 1 value to be true, got false")
	}

	// Set the value for "error 4" to true
	allowList.SetError("error 4")

	// Check that the value for "error 4" is not set to true
	if allowList.IsErrorSet("error 4") != false {
		t.Error("Expected error 4 value to be false, got true")
	}
}

// TestReset tests the Reset method of the AllowList data structure
func TestReset(t *testing.T) {
	// Create a new AllowList with 3 errors
	errors := []string{"error 1", "error 2", "error 3"}
	allowList := NewErrorAllowList(errors)

	// Set the value for "error 1" to true
	allowList.SetError("error 1")

	// Reset the values for all errors
	allowList.Reset()

	// Check that the values for all errors are set to false
	for _, err := range errors {
		if allowList.IsErrorSet(err) != false {
			t.Errorf("Expected %s value to be false, got true", err)
		}
	}
}

// TestNilPointer tests the AllowList data structure and its associated methods for nil pointer handling
func TestNilPointer(t *testing.T) {
	// Define a function to handle panic situations
	defer func() {
		if r := recover(); r != nil {
			t.Error("Unexpected panic occurred")
		}
	}()

	// Create a nil pointer for the AllowList data structure
	var nilAllowList *AllowList

	// Try to reset the values for all errors on null pointer
	nilAllowList.Reset()

	// Try to set the value for "error 1" to true on null pointer
	nilAllowList.SetError("error 1")

	// Try to check if error exists on null pointer
	if nilAllowList.IsErrorSet("error 1") != false {
		t.Error("Expected false, got true")
	}

}
