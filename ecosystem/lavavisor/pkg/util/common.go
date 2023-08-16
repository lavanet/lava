package lvutil

import (
	"archive/zip"
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
