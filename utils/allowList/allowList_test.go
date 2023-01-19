package allowList

import "testing"

// TestSetError tests the SetError method of the AllowList data structure
func TestSetError(t *testing.T) {
	// Create a new AllowList with 3 errors
	errors := []uint32{10, 11, 12}
	allowList := NewErrorAllowList(errors)

	// Set the value for error with a code 11 to true
	allowList.SetError(11)

	// Check that the value for error with a code 11 is set to true
	if allowList.IsErrorSet(11) != true {
		t.Error("Expected error with code 11 value to be true, got false")
	}

	// Set the value for error with a code 15 to true
	allowList.SetError(15)

	// Check that the value for error with code 15 is not set to true
	if allowList.IsErrorSet(15) != false {
		t.Error("Expected error with a code 15 to be false, got true")
	}
}

// TestReset tests the Reset method of the AllowList data structure
func TestReset(t *testing.T) {
	// Create a new AllowList with 3 errors
	errors := []uint32{10, 11, 12}
	allowList := NewErrorAllowList(errors)

	// Set the value for error with a code 11 to true
	allowList.SetError(11)

	// Reset the values for all errors
	allowList.Reset()

	// Check that the values for all errors are set to false
	for _, err := range errors {
		if allowList.IsErrorSet(err) != false {
			t.Errorf("Expected error with a code %d to be false, got true", err)
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

	// Try to set the value for error with a code 11 to true on null pointer
	nilAllowList.SetError(11)

	// Try to check if error exists on null pointer
	if nilAllowList.IsErrorSet(11) != false {
		t.Error("Expected false, got true")
	}

}
