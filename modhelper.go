package modhelper

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/solsw/oshelper"
)

// ModuleCache returns the [module cache] directory.
//
// [module cache]: https://go.dev/ref/mod#module-cache
func ModuleCache() (string, error) {
	// https://stackoverflow.com/questions/52126923/where-is-the-module-cache-in-golang
	goModCache := os.Getenv("GOMODCACHE")
	if goModCache != "" {
		return goModCache, nil
	}
	var modCache string
	goPath := os.Getenv("GOPATH")
	if goPath != "" {
		goPaths := strings.Split(goPath, string(filepath.ListSeparator))
		modCache = filepath.Join(goPaths[0], "pkg", "mod")
	} else {
		if runtime.GOOS == "linux" {
			home := ""
			sudoUser := os.Getenv("SUDO_USER")
			if sudoUser != "" {
				home = filepath.Join("/home", sudoUser)
			} else {
				home = os.Getenv("HOME")
			}
			modCache = filepath.Join(home, "go", "pkg", "mod")
		}
	}
	if modCache == "" {
		return "", errors.New("no module cache directory")
	}
	modCacheExists, err := oshelper.DirExists(modCache)
	if err != nil {
		return modCache, err
	}
	if !modCacheExists {
		return modCache, fmt.Errorf("no module cache directory '%s'", modCache)
	}
	return modCache, nil
}
