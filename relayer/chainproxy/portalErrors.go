package chainproxy

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
)

const (
	returnMaskedErrors = false
)

// input will be masked with a random GUID if returnMaskedErrors is set to true
// error contents will be saved in a dedicated log file located in:
// linux & darwin: /tmp/portal_guid_logs.log
// windows: current_path/portal_guid_logs.log
func GetUniqueGuidResponseForError(responseError error) string {
	guID := fmt.Sprintf("GUID%d", rand.Int63())
	var ret string
	rtos := runtime.GOOS
	var saveLogPath string
	switch rtos {
	case "windows":
		path, err := os.Getwd()
		if err != nil {
			return err.Error()
		}
		saveLogPath = filepath.Join(path, "portal_guid_logs.log")
	case "linux", "darwin":
		if _, err := os.Stat("/tmp"); os.IsNotExist(err) {
			return err.Error()
		}
		saveLogPath = "/tmp/portal_guid_logs.log"
	default:
		// other os isnt supported. return normal error format.
		return responseError.Error()
	}

	file, err := os.OpenFile(saveLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return responseError.Error()
	}
	defer file.Close()
	file.WriteString(fmt.Sprintf("%s: %s\n", guID, responseError.Error()))
	ret = "Error guid: " + guID
	if !returnMaskedErrors {
		ret += fmt.Sprintf("\nError: %v", responseError)
	}
	return ret
}
