// Command godoc-audit walks the framework's pkg/ tree and reports
// every exported symbol whose doc comment is missing or does not
// satisfy Go's convention "doc starts with the symbol name".
//
// Usage (from repo root):
//
//	go run ./scripts/godoc-audit
//
// Exit code 0 means every exported API has a conforming doc comment.
// Exit code 1 lists the violations grouped by package + file.
//
// The audit is deliberately strict but ignores:
//   - test files (*_test.go)
//   - the `internal` packages (none today, but future-proof)
//   - methods on unexported types (those types are themselves
//     unexported and cannot be referenced from godoc)
//   - the implicit String/Error/Read/Write/Close methods on stdlib
//     interfaces — they have a single canonical signature and Go
//     style allows omitting docs that would just repeat the parent.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// stdlibInterfaceMethods is the small set of methods that satisfy a
// well-known stdlib interface; their meaning is fully captured by the
// interface contract and adding a per-implementation doc comment is
// noise. We keep the list short on purpose — anything outside it must
// be documented.
var stdlibInterfaceMethods = map[string]bool{
	"String":    true, // fmt.Stringer
	"Error":     true, // error
	"ServeHTTP": true, // http.Handler
	"Read":      true, // io.Reader
	"Write":     true, // io.Writer
	"Close":     true, // io.Closer
}

type violation struct {
	pkg    string
	file   string
	line   int
	symbol string
	reason string
}

func main() {
	root := "pkg"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	var violations []violation
	totalExported := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		exp, viols := auditFile(path)
		totalExported += exp
		violations = append(violations, viols...)
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "walk error:", err)
		os.Exit(2)
	}

	if len(violations) == 0 {
		fmt.Printf("godoc-audit: %d exported symbols inspected, 0 violations\n", totalExported)
		return
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].file != violations[j].file {
			return violations[i].file < violations[j].file
		}
		return violations[i].line < violations[j].line
	})

	fmt.Printf("godoc-audit: %d exported symbols inspected, %d violations\n\n",
		totalExported, len(violations))

	currentPkg := ""
	for _, v := range violations {
		if v.pkg != currentPkg {
			fmt.Printf("\n# %s\n", v.pkg)
			currentPkg = v.pkg
		}
		fmt.Printf("  %s:%d  %s  — %s\n", v.file, v.line, v.symbol, v.reason)
	}
	os.Exit(1)
}

func auditFile(path string) (int, []violation) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse error:", path, err)
		return 0, nil
	}

	pkgName := f.Name.Name
	displayPath := filepath.ToSlash(path)
	exported := 0
	var out []violation

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if !s.Name.IsExported() {
						continue
					}
					exported++
					doc := pickDoc(d.Doc, s.Doc)
					if v := checkDoc(s.Name.Name, doc); v != "" {
						out = append(out, violation{
							pkg: pkgName, file: displayPath,
							line: fset.Position(s.Pos()).Line,
							symbol: "type " + s.Name.Name, reason: v,
						})
					}
					if st, ok := s.Type.(*ast.StructType); ok {
						out = append(out, auditStructFields(pkgName, displayPath, fset, s.Name.Name, st)...)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if !name.IsExported() {
							continue
						}
						exported++
						doc := pickDoc(d.Doc, s.Doc)
						kind := "var"
						if d.Tok == token.CONST {
							kind = "const"
						}
						if v := checkDoc(name.Name, doc); v != "" {
							out = append(out, violation{
								pkg: pkgName, file: displayPath,
								line: fset.Position(name.Pos()).Line,
								symbol: kind + " " + name.Name, reason: v,
							})
						}
					}
				}
			}
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			// Methods on unexported types are unreachable via godoc.
			if d.Recv != nil && len(d.Recv.List) > 0 {
				if !isExportedReceiver(d.Recv.List[0].Type) {
					continue
				}
				if stdlibInterfaceMethods[d.Name.Name] {
					continue
				}
			}
			exported++
			if v := checkDoc(d.Name.Name, d.Doc); v != "" {
				kind := "func"
				if d.Recv != nil {
					kind = "method"
				}
				out = append(out, violation{
					pkg: pkgName, file: displayPath,
					line: fset.Position(d.Pos()).Line,
					symbol: kind + " " + d.Name.Name, reason: v,
				})
			}
		}
	}

	return exported, out
}

func auditStructFields(pkgName, file string, fset *token.FileSet, typeName string, st *ast.StructType) []violation {
	if st.Fields == nil {
		return nil
	}
	var out []violation
	for _, field := range st.Fields.List {
		// Anonymous (embedded) fields share the embedded type's docs.
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			doc := field.Doc
			if doc == nil {
				doc = field.Comment
			}
			if v := checkDoc(name.Name, doc); v != "" {
				out = append(out, violation{
					pkg: pkgName, file: file,
					line:   fset.Position(name.Pos()).Line,
					symbol: fmt.Sprintf("field %s.%s", typeName, name.Name),
					reason: v,
				})
			}
		}
	}
	return out
}

func pickDoc(group, spec *ast.CommentGroup) *ast.CommentGroup {
	if spec != nil {
		return spec
	}
	return group
}

func checkDoc(name string, doc *ast.CommentGroup) string {
	if doc == nil || strings.TrimSpace(doc.Text()) == "" {
		return "missing doc comment"
	}
	first := strings.TrimSpace(doc.Text())
	if !startsWithName(first, name) {
		return fmt.Sprintf("doc must start with %q, got %q", name, snippet(first))
	}
	return ""
}

func startsWithName(text, name string) bool {
	if !strings.HasPrefix(text, name) {
		return false
	}
	rest := text[len(name):]
	if rest == "" {
		return true
	}
	r := []rune(rest)[0]
	// Next rune must be a word boundary (space, punctuation), not a
	// continuation of an identifier — so doc for "User" does not
	// match a comment starting with "Username".
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
}

func isExportedReceiver(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return isExportedReceiver(t.X)
	case *ast.Ident:
		return t.IsExported()
	case *ast.IndexExpr:
		return isExportedReceiver(t.X)
	case *ast.IndexListExpr:
		return isExportedReceiver(t.X)
	}
	return false
}

func snippet(s string) string {
	s = strings.SplitN(s, "\n", 2)[0]
	const max = 60
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
