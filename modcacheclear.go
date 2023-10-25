package modhelper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/solsw/oshelper"
	"github.com/solsw/semver"
)

// ModCacheClear clears Go [module cache].
// 'modPathPattern' - regular expression pattern (if not empty) to match [module path];
// 'versionsToKeep' - number of [module versions] (if greater than zero) to keep;
// 'removeAllVersions' - remove all [module versions];
// 'printSkipped' - print skipped [module path]s;
// 'w' is used (if not nil) to print text output.
//
// [module cache]: https://go.dev/ref/mod#module-cache
// [module path]: https://go.dev/ref/mod#module-path
// [module versions]: https://go.dev/ref/mod#versions
func ModCacheClear(modPathPattern string, versionsToKeep int, removeAllVersions bool,
	printSkipped bool, w io.Writer) error {
	modDir, err := ModuleCache()
	if err != nil {
		return err
	}
	var reModPath *regexp.Regexp
	if modPathPattern != "" {
		reModPath, err = regexp.Compile(modPathPattern)
		if err != nil {
			return err
		}
	}
	if w != nil {
		fmt.Fprintln(w, "module cache\n\t"+modDir)
	}
	mapModPathOsPaths, err := getMapModPathOsPaths(modDir)
	if err != nil {
		return err
	}
	modCacheModules := getModCacheModules(mapModPathOsPaths)
	err = modCacheClearPrim(reModPath, modCacheModules, versionsToKeep, removeAllVersions, printSkipped, w)
	if err != nil {
		return err
	}
	return nil
}

func modCacheClearPrim(reModPath *regexp.Regexp, modCacheModules []tModCacheModule,
	versionsToKeep int, removeAllVersions bool, printSkipped bool, w io.Writer) error {
	for _, module := range modCacheModules {
		if reModPath != nil && !reModPath.MatchString(string(module.modPath)) {
			if printSkipped && w != nil {
				fmt.Fprintln(w, string(module.modPath)+"\n\t**** SKIPPED ****")
			}
			continue
		}
		if w != nil {
			fmt.Fprintln(w, string(module.modPath))
		}
		vers := module.modVersions
		if len(vers) == 0 {
			if w != nil {
				fmt.Fprintln(w, "\t**** NO VERSIONS ****")
			}
			continue
		}
		for i, ver := range vers {
			if w != nil {
				fmt.Fprint(w, "\t"+string(ver.osPath))
			}
			if removeAllVersions || (versionsToKeep > 0 && i < len(vers)-versionsToKeep) {
				if err := os.RemoveAll(string(ver.osPath)); err != nil {
					return err
				}
				if w != nil {
					fmt.Fprint(w, " - deleted")
				}
				parentDir := filepath.Dir(string(ver.osPath))
				for filepath.Base(parentDir) != "mod" {
					if err := os.Remove(parentDir); err != nil {
						// parentDir is not empty
						break
					}
					if w != nil {
						fmt.Fprint(w, "\n\t"+parentDir+" - deleted")
					}
					parentDir = filepath.Dir(parentDir)
				}
			}
			if w != nil {
				fmt.Fprintln(w)
			}
		}
	}
	return nil
}

// helper types to explicitly specify what entities these strings denote
type (
	tModPath string
	tOsPath  string
)

// returns map with key->module path, element->OS paths of different versions of that module
func getMapModPathOsPaths(modDir string) (map[tModPath][]tOsPath, error) {
	m := make(map[tModPath][]tOsPath)
	if err := mapModPathOsPathsFromDir(modDir, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func mapModPathOsPathsFromDir(dir string, m *map[tModPath][]tOsPath) error {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		innerDir := filepath.Join(dir, dirEntry.Name())
		goModPath := filepath.Join(innerDir, "go.mod")
		goModExists, err := oshelper.FileExists(goModPath)
		if err != nil {
			return err
		}
		if goModExists {
			if mp, err := ModulePathFromGoMod(goModPath); err == nil {
				modPath := tModPath(mp)
				(*m)[modPath] = append((*m)[modPath], tOsPath(innerDir))
			}
		}
		if err := mapModPathOsPathsFromDir(innerDir, m); err != nil {
			return err
		}
	}
	return nil
}

// tModCacheModule contains information about module from [module cache]:
// [module path] and (OS path and SemVer) of all its versions.
//
// [module cache]: https://go.dev/ref/mod#module-cache
// [module path]: https://go.dev/ref/mod#module-path
type tModCacheModule struct {
	modPath     tModPath
	modVersions []struct {
		osPath tOsPath
		semVer semver.SemVer
	}
}

// getModCacheModules returns all modules with valid SemVer from module cache
func getModCacheModules(mapModPathOsPaths map[tModPath][]tOsPath) []tModCacheModule {
	var modCacheModules []tModCacheModule
	for modPath, osPaths := range mapModPathOsPaths {
		modCacheModule := tModCacheModule{modPath: modPath}
		for _, osPath := range osPaths {
			semVer, err := SemVerFromDirPath(string(osPath))
			if err != nil {
				continue
			}
			modCacheModule.modVersions = append(modCacheModule.modVersions,
				struct {
					osPath tOsPath
					semVer semver.SemVer
				}{osPath, semVer})
		}
		sort.Slice(modCacheModule.modVersions, func(i, j int) bool {
			return semver.Less(modCacheModule.modVersions[i].semVer, modCacheModule.modVersions[j].semVer)
		})
		modCacheModules = append(modCacheModules, modCacheModule)
	}
	sort.Slice(modCacheModules, func(i, j int) bool {
		return modCacheModules[i].modPath < modCacheModules[j].modPath
	})
	return modCacheModules
}
