package modhelper

import (
	"errors"
	"strings"
)

// https://go.dev/ref/mod#go-mod-file-ident
func validPathElem(pathElem string) error {
	// Each path element is a non-empty string
	if pathElem == "" {
		return errors.New("empty path element")
	}
	// A path element may not begin or end with a dot (., U+002E).
	if strings.HasPrefix(pathElem, ".") || strings.HasSuffix(pathElem, ".") {
		return errors.New("path element begins or ends with a dot: " + pathElem)
	}
	// made of up ASCII letters, ASCII digits, and limited ASCII punctuation (-, ., _, and ~).
	for _, r := range pathElem {
		if !(('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || ('0' <= r && r <= '9') ||
			r == '-' || r == '.' || r == '_' || r == '~') {
			return errors.New("path element contains invalid character: " + string(r))
		}
	}
	// The element prefix up to the first dot must not be a reserved file name on Windows, regardless of case (CON, com1, NuL, and so on).
	prefix, _, _ := strings.Cut(pathElem, ".")
	switch strings.ToUpper(prefix) {
	case "CON", "PRN", "AUX", "NUL",
		"COM0", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT0", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return errors.New("path element prefix must not be a reserved file name on Windows: " + prefix)
	}
	// The element prefix up to the first dot must not end with a tilde followed by one or more digits (like EXAMPL~1.COM).
	_, after, found := strings.Cut(prefix, "~")
	if found {
		for _, r := range after {
			if '0' <= r && r <= '9' {
				return errors.New("path element prefix must not end with a tilde followed by one or more digits: " + prefix)
			}
		}
	}
	return nil
}

// ValidModulePath checks if the [module path] is [valid].
//
// [module path]: https://go.dev/ref/mod#module-path
// [valid]: https://go.dev/ref/mod#go-mod-file-ident
func ValidModulePath(modPath string) error {
	if modPath == "" {
		return errors.New("empty module path")
	}
	// It must not begin or end with a slash.
	if strings.HasPrefix(modPath, "/") || strings.HasSuffix(modPath, "/") {
		return errors.New("module path begins or ends with a slash: " + modPath)
	}
	// The path must consist of one or more path elements separated by slashes (/, U+002F).
	ee := strings.Split(modPath, "/")
	for _, e := range ee {
		if err := validPathElem(e); err != nil {
			return err
		}
	}
	return nil
}
