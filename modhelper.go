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
	"github.com/solsw/semver"
)

// ModuleCache returns the [module cache] directory.
//
// [module cache]: https://go.dev/ref/mod#module-cache
func ModuleCache() (string, error) {
	// https://stackoverflow.com/questions/52126923/where-is-the-module-cache-in-golang
	var modCache string
	if goModCache := os.Getenv("GOMODCACHE"); goModCache != "" {
		modCache = goModCache
	} else if goPath := os.Getenv("GOPATH"); goPath != "" {
		goPaths := strings.SplitN(goPath, string(filepath.ListSeparator), 2)
		modCache = filepath.Join(goPaths[0], "pkg", "mod")
	} else {
		// GOPATH is unset, so Go defaults it to $HOME/go (%USERPROFILE%\go on
		// Windows) on every platform.
		var home string
		// Under sudo on Linux, HOME may point at root rather than the invoking
		// user, so prefer that user's home explicitly.
		if runtime.GOOS == "linux" {
			if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
				home = filepath.Join("/home", sudoUser)
			}
		}
		if home == "" {
			h, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			home = h
		}
		modCache = filepath.Join(home, "go", "pkg", "mod")
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
			// A quoted module path is unquoted; anything else (including an
			// unquote failure) is used verbatim.
			if unquoted, er2 := strconv.Unquote(modPath); er2 == nil {
				modPath = unquoted
			}
			return modPath, ValidModulePath(modPath)
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no module path in '" + gomod + "'")
}

// SemVerFromDirPath extracts [semver.SemVer] from the directory path.
func SemVerFromDirPath(dirPath string) (semver.SemVer, error) {
	_, after, found := strings.Cut(dirPath, "@v")
	if !found {
		return semver.SemVer{}, fmt.Errorf("no SemVer in directory path '%s'", dirPath)
	}
	return semver.Parse(after)
}
