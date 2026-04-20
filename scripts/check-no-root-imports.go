// check-no-root-imports verifies that the framework module never depends on
// any package outside `pkg/`. It is intentionally simple: walk every .go
// file in the repository root (excluding examples/ subprojects, which have
// their own go.mod), parse the imports, and reject any import that starts
// with the framework module path but does not point into pkg/.
package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const moduleRoot = "github.com/fastygo/framework"

var allowedFrameworkPrefixes = []string{
	moduleRoot,
	moduleRoot + "/pkg",
}

func main() {
	fset := token.NewFileSet()
	violations := 0

	visit := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			// Skip examples/* — they are independent Go modules with their
			// own composition roots and are allowed to import everything.
			if path != "." && name == "examples" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			// Skip unparseable files — they are typically generated stubs
			// (e.g. _templ.go before `templ generate` runs). Surface in
			// stderr so a real syntax error is still visible.
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", path, parseErr)
			return nil
		}

		for _, importSpec := range file.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				continue
			}
			if !strings.HasPrefix(importPath, moduleRoot) {
				continue
			}
			if isAllowed(importPath) {
				continue
			}
			position := fset.Position(importSpec.Path.Pos())
			fmt.Printf("%s:%d: %s (only %s/pkg/... is allowed inside the framework module)\n", position.Filename, position.Line, importPath, moduleRoot)
			violations++
		}
		return nil
	}

	if err := filepath.WalkDir(".", visit); err != nil {
		fmt.Printf("failed to scan repository: %v\n", err)
		os.Exit(1)
	}

	if violations > 0 {
		fmt.Printf("Failing due to %d framework boundary violation(s).\n", violations)
		os.Exit(1)
	}
	fmt.Println("No-root imports check passed: framework module only references pkg/...")
}

func isAllowed(importPath string) bool {
	for _, prefix := range allowedFrameworkPrefixes {
		if importPath == prefix {
			return true
		}
		if strings.HasPrefix(importPath, prefix+"/") {
			return true
		}
	}
	return false
}
