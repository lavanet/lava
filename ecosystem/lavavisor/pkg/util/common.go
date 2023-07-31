package lvutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lavanet/lava/utils"
)

func ExpandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", utils.LavaFormatError("cannot get user home directory", err)
	}
	return filepath.Join(home, path[1:]), nil
}
