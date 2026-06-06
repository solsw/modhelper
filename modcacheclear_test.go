package modhelper

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// Test_forceRemoveAll verifies that forceRemoveAll removes a tree whose files
// and directories carry the read-only permissions Go sets on module cache
// entries (files 0444, directories 0555), which plain os.RemoveAll cannot.
func Test_forceRemoveAll(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "github.com", "solsw", "modhelper@v1.2.3")
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := []string{
		filepath.Join(target, "go.mod"),
		filepath.Join(target, "sub", "file.go"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Apply read-only module-cache permissions, deepest entries first.
	roDirs := []string{filepath.Join(target, "sub"), target}
	for _, f := range files {
		if err := os.Chmod(f, 0o444); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range roDirs {
		if err := os.Chmod(d, 0o555); err != nil {
			t.Fatal(err)
		}
	}

	if err := forceRemoveAll(target); err != nil {
		t.Fatalf("forceRemoveAll() error = %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target still exists, stat err = %v", err)
	}
}

// Test_mapModPathOsPathsFromDir verifies that the walker records the versioned
// module directory and does not descend into its source tree (a nested go.mod
// inside the extracted source must not produce an extra entry).
func Test_mapModPathOsPathsFromDir(t *testing.T) {
	modDir := t.TempDir()
	modVer := filepath.Join(modDir, "github.com", "solsw", "modhelper@v1.2.3")
	nested := filepath.Join(modVer, "internal", "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	// go.mod at the module root (should be recorded) ...
	if err := os.WriteFile(filepath.Join(modVer, "go.mod"),
		[]byte("module github.com/solsw/modhelper\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// ... and a stray go.mod deeper in the source tree (must be ignored).
	if err := os.WriteFile(filepath.Join(nested, "go.mod"),
		[]byte("module github.com/solsw/modhelper/internal/nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := make(map[tModPath][]tOsPath)
	if err := mapModPathOsPathsFromDir(modDir, modDir, &m); err != nil {
		t.Fatalf("mapModPathOsPathsFromDir() error = %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d module paths, want 1: %v", len(m), m)
	}
	got := m["github.com/solsw/modhelper"]
	if len(got) != 1 || string(got[0]) != modVer {
		t.Fatalf("got osPaths %v, want [%s]", got, modVer)
	}
}

// Test_mapModPathOsPathsFromDir_cacheRoot verifies that the "cache" directory at
// the module cache root is skipped (it holds download artifacts, not extracted
// modules), while a "cache" directory that is a genuine module-path element
// deeper in the tree is still traversed.
func Test_mapModPathOsPathsFromDir_cacheRoot(t *testing.T) {
	modDir := t.TempDir()
	// Looks like an extracted module but sits under the root "cache" dir: skip.
	underCache := filepath.Join(modDir, "cache", "download", "github.com", "foo@v1.0.0")
	if err := os.MkdirAll(underCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(underCache, "go.mod"),
		[]byte("module github.com/foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// "cache" as a real module-path element, deeper in the tree: must be walked.
	deep := mkVersion(t, modDir, "github.com/x/cache/inner", "v2.0.0")

	m := make(map[tModPath][]tOsPath)
	if err := mapModPathOsPathsFromDir(modDir, modDir, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["github.com/foo"]; ok {
		t.Errorf("module under root cache dir was recorded: %v", m)
	}
	got := m["github.com/x/cache/inner"]
	if len(got) != 1 || string(got[0]) != deep {
		t.Errorf("deep cache-element module = %v, want [%s]", got, deep)
	}
}

// Test_getModCacheModules verifies that modules are sorted by module path, their
// versions are sorted by SemVer (numerically, not lexically), directory paths
// with an unparsable version are dropped, and a module whose versions are all
// unparsable is still reported (with no versions).
func Test_getModCacheModules(t *testing.T) {
	m := map[tModPath][]tOsPath{
		// out-of-order and not lexically sortable: v1.10.0 > v1.2.0 numerically
		// but sorts before it as a string; "@vbad" has no valid SemVer.
		"github.com/a/one": {"/x/one@v1.10.0", "/x/one@v1.2.0", "/x/one@vbad", "/x/one@v1.3.0"},
		"github.com/b/two": {"/x/two@v0.1.0"},
		"github.com/c/non": {"/x/non@vbad"},
	}
	got := getModCacheModules(m)

	wantPaths := []tModPath{"github.com/a/one", "github.com/b/two", "github.com/c/non"}
	if len(got) != len(wantPaths) {
		t.Fatalf("got %d modules, want %d: %v", len(got), len(wantPaths), got)
	}
	for i, w := range wantPaths {
		if got[i].modPath != w {
			t.Errorf("module[%d] = %q, want %q", i, got[i].modPath, w)
		}
	}

	// "one": "@vbad" dropped, the rest sorted ascending by SemVer.
	var oneVers []string
	for _, v := range got[0].modVersions {
		oneVers = append(oneVers, string(v.osPath))
	}
	wantOne := []string{"/x/one@v1.2.0", "/x/one@v1.3.0", "/x/one@v1.10.0"}
	if !reflect.DeepEqual(oneVers, wantOne) {
		t.Errorf("one versions = %v, want %v", oneVers, wantOne)
	}

	// "non": only version is unparsable, so it is reported with no versions.
	if n := len(got[2].modVersions); n != 0 {
		t.Errorf("non versions = %d, want 0", n)
	}
}

// mkVersion creates an extracted-module directory modDir/<modPath>@<ver> holding
// a go.mod with the given module path, mirroring the module cache layout, and
// returns its path. ver includes the leading "v" (e.g. "v1.2.3").
func mkVersion(t *testing.T, modDir, modPath, ver string) string {
	t.Helper()
	elems := strings.Split(modPath, "/")
	last := elems[len(elems)-1]
	parent := filepath.Join(append([]string{modDir}, elems[:len(elems)-1]...)...)
	dir := filepath.Join(parent, last+"@"+ver)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module "+modPath+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
	return err == nil
}

// Test_modCacheClearPrim drives the deletion/selection logic against a real
// module-cache-shaped temp tree: keep-newest selection, removeAllVersions
// precedence, the module-path regexp skip, and the no-op default.
func Test_modCacheClearPrim(t *testing.T) {
	t.Run("keep newest N", func(t *testing.T) {
		modDir := t.TempDir()
		v1 := mkVersion(t, modDir, "github.com/a/one", "v1.2.0")
		v2 := mkVersion(t, modDir, "github.com/a/one", "v1.10.0")
		v3 := mkVersion(t, modDir, "github.com/a/one", "v1.3.0")
		two := mkVersion(t, modDir, "github.com/b/two", "v0.1.0")

		modules := loadModules(t, modDir)
		var buf bytes.Buffer
		if err := modCacheClearPrim(modDir, nil, modules, 1, false, false, &buf); err != nil {
			t.Fatal(err)
		}
		// one: keep newest (v1.10.0), delete v1.2.0 and v1.3.0.
		assertGone(t, v1, v3)
		assertKept(t, v2)
		// two: only one version, keep=1 keeps it.
		assertKept(t, two)
	})

	t.Run("removeAllVersions overrides keep", func(t *testing.T) {
		modDir := t.TempDir()
		v1 := mkVersion(t, modDir, "github.com/a/one", "v1.0.0")
		v2 := mkVersion(t, modDir, "github.com/a/one", "v2.0.0")

		modules := loadModules(t, modDir)
		var buf bytes.Buffer
		// versionsToKeep would keep one, but removeAllVersions wins.
		if err := modCacheClearPrim(modDir, nil, modules, 5, true, false, &buf); err != nil {
			t.Fatal(err)
		}
		assertGone(t, v1, v2)
		// The now-empty module subtree is cleaned up to modDir.
		assertGone(t, filepath.Join(modDir, "github.com", "a"))
	})

	t.Run("regexp skip", func(t *testing.T) {
		modDir := t.TempDir()
		one := mkVersion(t, modDir, "github.com/a/one", "v1.0.0")
		two := mkVersion(t, modDir, "github.com/b/two", "v1.0.0")

		modules := loadModules(t, modDir)
		var buf bytes.Buffer
		re := regexp.MustCompile("two")
		if err := modCacheClearPrim(modDir, re, modules, 0, true, true, &buf); err != nil {
			t.Fatal(err)
		}
		// Only "two" matches the pattern and is removed; "one" is skipped.
		assertGone(t, two)
		assertKept(t, one)
		if out := buf.String(); !strings.Contains(out, "SKIPPED") ||
			!strings.Contains(out, "github.com/a/one") {
			t.Errorf("skip not reported in output:\n%s", out)
		}
	})

	t.Run("no-op when keep is zero", func(t *testing.T) {
		modDir := t.TempDir()
		v1 := mkVersion(t, modDir, "github.com/a/one", "v1.0.0")
		v2 := mkVersion(t, modDir, "github.com/a/one", "v2.0.0")

		modules := loadModules(t, modDir)
		var buf bytes.Buffer
		if err := modCacheClearPrim(modDir, nil, modules, 0, false, false, &buf); err != nil {
			t.Fatal(err)
		}
		assertKept(t, v1, v2)
	})
}

func loadModules(t *testing.T, modDir string) []tModCacheModule {
	t.Helper()
	m := make(map[tModPath][]tOsPath)
	if err := mapModPathOsPathsFromDir(modDir, modDir, &m); err != nil {
		t.Fatal(err)
	}
	return getModCacheModules(m)
}

func assertGone(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if exists(t, p) {
			t.Errorf("expected removed, still present: %s", p)
		}
	}
}

func assertKept(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if !exists(t, p) {
			t.Errorf("expected kept, missing: %s", p)
		}
	}
}
