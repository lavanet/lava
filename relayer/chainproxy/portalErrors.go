package chainproxy

import (
	"fmt"
	"math/rand"
)

const (
	returnMaskedErrors = false
)

// input will be masked with a random GUID if returnMaskedErrors is set to true
func GetUniqueGuidResponseForError(responseError error) string {
	guID := fmt.Sprintf("GUID%d", rand.Int63())
	var ret string

	ret = "Error guid: " + guID
	if !returnMaskedErrors {
		ret += fmt.Sprintf("\nError: %v", responseError)
	}
	return ret
}
