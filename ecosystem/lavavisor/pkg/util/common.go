package lvutil

import (
	"archive/zip"
	"fmt"
	"io"
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

func Unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, utils.LavaFormatError("illegal file path", nil)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

// Parse the version string "vX.Y.Z" into a struct with X, Y, Z as integers
func ParseVersion(version string) (v struct {
	Major, Minor, Patch int
}) {
	fmt.Sscanf(version, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	return
}

// Decrement the version. If Patch is 0, decrement Minor and reset Patch to 9. If Minor is 0, decrement Major.
func DecrementVersion(v *struct {
	Major, Minor, Patch int
}) {
	if v.Patch > 0 {
		v.Patch--
	} else if v.Minor > 0 {
		v.Minor--
		v.Patch = 9
	}
}

// Check if version v1 is less than v2
func IsVersionLessThan(v1, v2 *struct {
	Major, Minor, Patch int
}) bool {
	if v1.Major < v2.Major {
		return true
	}
	if v1.Major == v2.Major && v1.Minor < v2.Minor {
		return true
	}
	if v1.Major == v2.Major && v1.Minor == v2.Minor && v1.Patch < v2.Patch {
		return true
	}
	return false
}

// Format the version struct back to a string "vX.Y.Z"
func FormatVersion(v *struct {
	Major, Minor, Patch int
}) string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
