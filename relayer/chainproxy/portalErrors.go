package chainproxy

import (
	"fmt"
	"math/rand"

	"github.com/lavanet/lava/utils"
)

var ReturnMaskedErrors = "false"

// input will be masked with a random GUID if returnMaskedErrors is set to true
func GetUniqueGuidResponseForError(responseError error) string {
	guID := fmt.Sprintf("GUID%d", rand.Int63())
	var ret string
	ret = "Error GUID: " + guID
	utils.LavaFormatError("UniqueGuidResponseForError", responseError, &map[string]string{"GUID": guID})
	if ReturnMaskedErrors == "false" {
		ret += fmt.Sprintf(", Error: %v", responseError)
	}
	return ret
}
