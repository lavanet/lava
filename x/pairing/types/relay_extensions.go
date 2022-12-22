package types

// ShallowCopy makes a shallow copy of the relay request, and returns it
// A shallow copy includes the values of all fields in the original struct,
// but any nested values (such as slices, maps, and pointers) are shared between the original and the copy.
func (m *RelayRequest) ShallowCopy() *RelayRequest {
	if m == nil {
		return nil
	}

	requestCopy := *m
	return &requestCopy
}
