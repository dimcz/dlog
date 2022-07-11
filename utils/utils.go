package utils

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/dimcz/dlog/logging"
)

func Check(e error) {
	if e != nil {
		panic(e)
	}
}

//goland:noinspection GoUnusedExportedFunction
func Max(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func Min(a, b int) int {
	if a > b {
		return b
	}

	return a
}

//goland:noinspection GoUnusedExportedFunction
func Max64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func Min64(a, b int64) int64 {
	if a > b {
		return b
	}

	return b
}

func OpenRewrite(path string) *os.File {
	var (
		err error
		f   *os.File
	)

	openFile := func() error {
		f, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)

		return err
	}

	if err = openFile(); os.IsExist(err) {
		logging.LogOnErr(os.Remove(path))
		err = openFile()
	}

	Check(err)

	return f
}

func ValidateRegularFile(filename string) error {
	fi, err := os.Stat(filename)

	switch {
	case os.IsNotExist(err):
		return errors.New(filename + ": No such file or directory")
	case os.IsPermission(err):
		return errors.New(filename + ": Permission denied")
	case err != nil:
		return err
	}

	switch fmode := fi.Mode(); {
	case fmode.IsDir():
		return errors.New(filename + " is a directory")
	case !fmode.IsRegular():
		return errors.New(filename + " is not a regular file")
	}

	return nil
}

func GetHomeDir() string {
	var homedir string

	currentUser, err := user.Current()

	if err != nil {
		homedir = os.Getenv("HOME")
		if homedir == "" {
			homedir = os.TempDir()
		}
	} else {
		homedir = currentUser.HomeDir
	}

	return homedir
}

func ExpandHomePath(path string) string {
	if len(path) < 2 || path[:2] != "~"+string(os.PathSeparator) {
		return path
	}

	return filepath.Join(GetHomeDir(), path[2:])
}

func ExitOnErr(err error) {
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func MakeCacheFile() (f *os.File, err error) {
	return ioutil.TempFile(os.TempDir(), "dlog_")
}
