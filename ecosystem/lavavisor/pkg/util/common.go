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

func GetLavavisorPath(dir string) (lavavisorPath string, err error) {
	dir, err = ExpandTilde(dir)
	if err != nil {
		return "", utils.LavaFormatError("unable to expand directory path", err)
	}
	// Build path to ./lavavisor
	lavavisorPath = filepath.Join(dir, "./.lavavisor")

	// Check if ./lavavisor directory exists
	if _, err := os.Stat(lavavisorPath); os.IsNotExist(err) {
		// ToDo: handle case where user didn't set up the file
		return "", utils.LavaFormatError("lavavisor directory does not exist at path", err, utils.Attribute{Key: "lavavisorPath", Value: lavavisorPath})
	}

	return lavavisorPath, nil
}

func Copy(src, dest string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return utils.LavaFormatError("couldn't read source file", err)
	}

	err = os.WriteFile(dest, input, 0o755)
	if err != nil {
		return utils.LavaFormatError("couldn't write destination file", err)
	}
	return nil
}
