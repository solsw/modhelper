# modhelper
[![Go Reference](https://pkg.go.dev/badge/github.com/solsw/modhelper.svg)](https://pkg.go.dev/github.com/solsw/modhelper)
[![GitHub](https://img.shields.io/badge/github--green?logo=github)](https://github.com/solsw/modhelper)

Helpers for [Go Modules](https://go.dev/ref/mod).

## Installation

```sh
go get github.com/solsw/modhelper
```

```go
import "github.com/solsw/modhelper"
```

## Overview

`modhelper` provides utilities for working with the local
[module cache](https://go.dev/ref/mod#module-cache) and with
[module paths](https://go.dev/ref/mod#module-path): locating the cache,
clearing cached module versions, reading the module path from a `go.mod`
file, validating module paths, and extracting a `SemVer` from a directory
path.

## API

### `ModuleCache() (string, error)`

Returns the [module cache](https://go.dev/ref/mod#module-cache) directory.

The directory is resolved in the following order:

1. `GOMODCACHE`, if set.
2. The first entry of `GOPATH` joined with `pkg/mod`, if `GOPATH` is set.
3. `$HOME/go/pkg/mod` (`%USERPROFILE%\go\pkg\mod` on Windows) otherwise. On
   Linux running under `sudo`, the invoking user's home (`/home/$SUDO_USER`)
   is preferred over `root`'s.

An error is returned if the resolved directory does not exist.

```go
dir, err := modhelper.ModuleCache()
if err != nil {
	log.Fatal(err)
}
fmt.Println("module cache:", dir)
```

### `ModCacheClear(modPathPattern string, versionsToKeep int, removeAllVersions bool, dryRun bool, printSkipped bool, w io.Writer) error`

Clears Go's module cache. Within each module path, cached versions are sorted
oldest-to-newest by SemVer before any deletion decision is made. Read-only
permissions that Go sets on cache entries are relaxed automatically, and
parent directories that become empty are removed up to (but not including) the
cache root.

| Parameter           | Description                                                                                                                                            |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `modPathPattern`    | Regular expression matched against each module path. Empty matches every module.                                                                      |
| `versionsToKeep`    | Number of newest versions to keep per module path. Has effect only when greater than zero; zero or negative keeps every version (deletes nothing).    |
| `removeAllVersions` | Remove all versions. Takes precedence over `versionsToKeep`.                                                                                          |
| `dryRun`            | Only report what would be deleted, without deleting anything.                                                                                         |
| `printSkipped`      | Print module paths skipped by `modPathPattern`.                                                                                                       |
| `w`                 | Destination for text output. Pass `nil` to suppress output.                                                                                           |

```go
// Keep the 2 newest versions of every cached module, reporting to stdout.
err := modhelper.ModCacheClear("", 2, false, false, false, os.Stdout)
if err != nil {
	log.Fatal(err)
}
```

```go
// Preview removal of all cached versions of modules under example.com.
err := modhelper.ModCacheClear(`^example\.com/`, 0, true, true, false, os.Stdout)
```

### `ModulePathFromGoMod(gomod string) (string, error)`

Reads the [module path](https://go.dev/ref/mod#module-path) from the `module`
directive of the given `go.mod` file. A quoted module path is unquoted. The
returned path is validated with `ValidModulePath`.

```go
path, err := modhelper.ModulePathFromGoMod("go.mod")
if err != nil {
	log.Fatal(err)
}
fmt.Println(path) // e.g. github.com/solsw/modhelper
```

### `ValidModulePath(modPath string) error`

Reports whether `modPath` is a [valid](https://go.dev/ref/mod#go-mod-file-ident)
module path, returning `nil` when valid and a descriptive error otherwise.
Enforces the Go module path rules: non-empty, no leading or trailing slash,
each `/`-separated element made of ASCII letters, digits and `-`, `.`, `_`,
`~`, not beginning or ending with a dot, not a Windows reserved name, and not
ending with a tilde-plus-digits short-name form.

```go
if err := modhelper.ValidModulePath("example.com/foo"); err != nil {
	log.Fatal(err)
}
```

### `SemVerFromDirPath(dirPath string) (semver.SemVer, error)`

Extracts a [`semver.SemVer`](https://pkg.go.dev/github.com/solsw/semver) from a
directory path by parsing the portion after `@v`.

```go
v, err := modhelper.SemVerFromDirPath("example.com/foo@v1.2.3")
if err != nil {
	log.Fatal(err)
}
fmt.Println(v) // 1.2.3
```
