package modhelper

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

// ModulePathFromGoMod retrieves the [module path] from the [go.mod] file.
//
// [module path]: https://go.dev/ref/mod#module-path
// [go.mod]: https://go.dev/ref/mod#go-mod-file
func ModulePathFromGoMod(gomod string) (string, error) {
	f, err := os.Open(gomod)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var mod, modPath string
		_, er1 := fmt.Sscan(sc.Text(), &mod, &modPath)
		if er1 == nil {
			if mod != "module" {
				continue
			}
			unquotedModulePath, er2 := strconv.Unquote(modPath)
			if er2 == strconv.ErrSyntax {
				return modPath, ValidModulePath(modPath)
			}
			return unquotedModulePath, ValidModulePath(unquotedModulePath)
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no module path in '" + gomod + "'")
}
