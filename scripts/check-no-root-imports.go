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

type forbiddenPath struct {
	name  string
	prefix string
}

func main() {
	moduleRoot := "github.com/fastygo/framework"
	forbidden := []forbiddenPath{
		{name: "internal/features", prefix: moduleRoot + "/internal/features"},
		{name: "internal/site/features", prefix: moduleRoot + "/internal/site/features"},
		{name: "views", prefix: moduleRoot + "/views"},
		{name: "fixtures", prefix: moduleRoot + "/fixtures"},
	}

	fset := token.NewFileSet()
	found := false

	visit := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}

		for _, importSpec := range file.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil {
				continue
			}
			for _, rule := range forbidden {
				if isForbiddenImport(importPath, rule.prefix) {
					position := fset.Position(importSpec.Path.Pos())
					fmt.Printf("%s:%d: %s (%s)\n", position.Filename, position.Line, importPath, rule.name)
					found = true
				}
			}
		}
		return nil
	}

	if err := filepath.WalkDir(".", visit); err != nil {
		fmt.Printf("failed to scan repository: %v\n", err)
		os.Exit(1)
	}

	if found {
		fmt.Println("Failing due to root-import policy violations.")
		os.Exit(1)
	}

	fmt.Println("No-root imports check passed.")
}

func isForbiddenImport(importPath string, prefix string) bool {
	return importPath == prefix || strings.HasPrefix(importPath, prefix+"/")
}
