package modhelper

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/solsw/oshelper"
	"github.com/solsw/semver"
)

// ModCacheClear clears Go's [module cache].
// 'modPathPattern' - regular expression pattern (if not empty) to match [module path];
// 'versionsToKeep' - number of newest [module versions] to keep per [module path];
// it has effect only when greater than zero, and zero (or negative) keeps every
// version (deletes nothing) - use 'removeAllVersions' to delete all versions;
// 'removeAllVersions' - remove all [module versions] (takes precedence over 'versionsToKeep');
// 'dryRun' - only report what would be deleted, without deleting anything;
// 'printSkipped' - print skipped [module path]s;
// 'w' is used (if not nil) to print text output.
//
// [module cache]: https://go.dev/ref/mod#module-cache
// [module path]: https://go.dev/ref/mod#module-path
// [module versions]: https://go.dev/ref/mod#versions
func ModCacheClear(modPathPattern string, versionsToKeep int, removeAllVersions bool,
	dryRun bool, printSkipped bool, w io.Writer) error {
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
	err = modCacheClearPrim(modDir, reModPath, modCacheModules, versionsToKeep, removeAllVersions, dryRun, printSkipped, w)
	if err != nil {
		return err
	}
	return nil
}

func modCacheClearPrim(modDir string, reModPath *regexp.Regexp, modCacheModules []tModCacheModule,
	versionsToKeep int, removeAllVersions bool, dryRun bool, printSkipped bool, w io.Writer) error {
	// modDir is the boundary for upward empty-directory cleanup;
	// clean it so it can be compared with paths produced by filepath.Join/filepath.Dir.
	modDir = filepath.Clean(modDir)
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
				if dryRun {
					if w != nil {
						fmt.Fprint(w, " - would be deleted")
					}
				} else {
					if err := forceRemoveAll(string(ver.osPath)); err != nil {
						return err
					}
					if w != nil {
						fmt.Fprint(w, " - deleted")
					}
					// Remove now-empty parent directories, walking up no further than modDir.
					parentDir := filepath.Dir(string(ver.osPath))
					for parentDir != modDir {
						// The grandparent must be writable to unlink parentDir from it.
						if err := makeWritable(filepath.Dir(parentDir)); err != nil {
							break
						}
						if err := os.Remove(parentDir); err != nil {
							// parentDir is not empty (or otherwise cannot be removed)
							break
						}
						if w != nil {
							fmt.Fprint(w, "\n\t"+parentDir+" - deleted")
						}
						parentDir = filepath.Dir(parentDir)
					}
				}
			}
			if w != nil {
				fmt.Fprintln(w)
			}
		}
	}
	return nil
}

// makeWritable adds owner write permission to path (and execute, for directories,
// so it can be traversed). Go marks module cache files 0444 and directories 0555,
// so they (and their entries) cannot be removed until permissions are relaxed.
func makeWritable(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	mode := info.Mode()
	add := fs.FileMode(0o200)
	if mode.IsDir() {
		add = 0o300
	}
	if mode.Perm()&add == add {
		return nil
	}
	return os.Chmod(path, mode.Perm()|add)
}

// forceRemoveAll removes path and everything under it, first clearing the
// read-only permissions that Go sets on module cache entries. Plain
// os.RemoveAll fails on such entries: on Unix removal needs write access on the
// containing directory, and on Windows read-only files reject deletion.
func forceRemoveAll(path string) error {
	// The parent must be writable so path itself can be unlinked.
	if parent := filepath.Dir(path); parent != path {
		if err := makeWritable(parent); err != nil {
			return err
		}
	}
	// Relax permissions on the whole subtree before removing it.
	err := filepath.WalkDir(path, func(p string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return makeWritable(p)
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

// helper types to explicitly specify what entities these strings denote
type (
	tModPath string
	tOsPath  string
)

// returns map with key->module path, element->OS paths of different versions of that module
func getMapModPathOsPaths(modDir string) (map[tModPath][]tOsPath, error) {
	m := make(map[tModPath][]tOsPath)
	if err := mapModPathOsPathsFromDir(modDir, modDir, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// mapModPathOsPathsFromDir walks dir (initially the module cache root, also
// passed as root) recording extracted modules into m.
func mapModPathOsPathsFromDir(root, dir string, m *map[tModPath][]tOsPath) error {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		// The "cache" directory directly under the module cache root holds only
		// download artifacts (zips, .info/.mod files) and the sumdb tree - never
		// extracted modules - so skip it to avoid walking that whole subtree. A
		// "cache" directory deeper down is a genuine module-path element.
		if dir == root && dirEntry.Name() == "cache" {
			continue
		}
		innerDir := filepath.Join(dir, dirEntry.Name())
		// An extracted module lives in a directory whose name embeds the version
		// (e.g. "modhelper@v1.2.3"). Such a directory is a module root: record it
		// and do not descend into its source tree, which contains no further
		// cached modules and would otherwise be walked file by file.
		if strings.Contains(dirEntry.Name(), "@") {
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
			continue
		}
		if err := mapModPathOsPathsFromDir(root, innerDir, m); err != nil {
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
