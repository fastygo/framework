// Command layout-audit validates custom site shell markup contracts.
//
// It is intentionally static and conservative: the audit reads .templ files
// and checks the ARIA/data attribute wiring used by UI8Kit shell behavior.
// It is meant for custom app shells that do not call ui8kit/layout.Shell.
//
// Usage (from repo root):
//
//	go run ./scripts/layout-audit ./examples/blog/internal/site/views
//
// Exit code 0 means every inspected custom shell contract is valid.
// Exit code 1 lists violations. Exit code 2 is reserved for runtime errors.
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type violation struct {
	file   string
	line   int
	rule   string
	reason string
}

type contractRegistry struct {
	Contracts map[string]contractDefinition `json:"contracts"`
}

type contractDefinition struct {
	Providers []contractProvider `json:"providers"`
}

type contractProvider struct {
	Component   string `json:"component"`
	TargetField string `json:"targetField"`
}

type auditOptions struct {
	contractsPath string
	roots         []string
}

var (
	idAttrRE     = regexp.MustCompile(`\bid\s*=\s*"([^"]+)"`)
	targetAttrRE = regexp.MustCompile(`\bdata-ui8kit-dialog-target\s*=\s*"([^"]+)"`)
	tagRE        = regexp.MustCompile(`(?s)<([a-zA-Z][a-zA-Z0-9-]*)([^>]*)>`)
	importRE     = regexp.MustCompile(`(?m)([A-Za-z_][A-Za-z0-9_]*)?\s*"([^"]+)"`)
)

func main() {
	opts := parseArgs(os.Args[1:])
	registry, err := loadContractRegistry(opts.contractsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "layout-audit:", err)
		os.Exit(2)
	}

	files, err := collectTemplFiles(opts.roots)
	if err != nil {
		fmt.Fprintln(os.Stderr, "layout-audit:", err)
		os.Exit(2)
	}

	var all []violation
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "layout-audit:", err)
			os.Exit(2)
		}
		all = append(all, auditFile(file, string(content), registry)...)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].file != all[j].file {
			return all[i].file < all[j].file
		}
		return all[i].line < all[j].line
	})

	if len(all) == 0 {
		fmt.Printf("layout-audit: %d templ files inspected, 0 violations\n", len(files))
		return
	}

	fmt.Printf("layout-audit: %d templ files inspected, %d violations\n\n", len(files), len(all))
	for _, v := range all {
		fmt.Printf("%s:%d %s %s\n", v.file, v.line, v.rule, v.reason)
	}
	os.Exit(1)
}

func parseArgs(args []string) auditOptions {
	opts := auditOptions{
		contractsPath: filepath.FromSlash("contracts/layout.contracts.json"),
		roots:         []string{"."},
	}
	roots := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--contracts":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "layout-audit: --contracts requires a path")
				os.Exit(2)
			}
			opts.contractsPath = args[i+1]
			i++
		default:
			roots = append(roots, arg)
		}
	}

	if len(roots) > 0 {
		opts.roots = roots
	}
	return opts
}

func loadContractRegistry(path string) (contractRegistry, error) {
	var registry contractRegistry
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return registry, nil
		}
		return registry, err
	}
	if err := json.Unmarshal(content, &registry); err != nil {
		return registry, fmt.Errorf("invalid contract registry %s: %w", path, err)
	}
	return registry, nil
}

func collectTemplFiles(roots []string) ([]string, error) {
	files := make([]string, 0)
	seen := map[string]bool{}

	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if strings.HasSuffix(root, ".templ") {
				files = appendIfNew(files, seen, root)
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", ".ui8px", "node_modules", "web", "dist", "coverage":
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".templ") {
				files = appendIfNew(files, seen, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(files)
	return files, nil
}

func appendIfNew(files []string, seen map[string]bool, file string) []string {
	clean := filepath.Clean(file)
	if seen[clean] {
		return files
	}
	seen[clean] = true
	return append(files, clean)
}

func auditFile(file, content string, registry contractRegistry) []violation {
	var out []violation
	providedContracts, providerTargets, providerViolations := detectContractProviders(file, content, registry)
	out = append(out, providerViolations...)

	ids := collectValues(idAttrRE, content)
	targets := collectValues(targetAttrRE, content)
	triggerTargets := map[string]bool{}
	closeTargets := map[string]bool{}
	dialogIDs := map[string]bool{}
	hasMain := strings.Contains(content, "<main")
	hasShellMarker := false

	for contractName, contractTargets := range providerTargets {
		for target := range contractTargets {
			targets[target] = true
			switch contractName {
			case "dialog.trigger":
				triggerTargets[target] = true
			case "dialog.close":
				closeTargets[target] = true
			}
		}
	}

	matches := tagRE.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		tagStart := match[0]
		tagText := content[match[0]:match[1]]
		attrs := content[match[4]:match[5]]
		line := lineAt(content, tagStart)

		if strings.Contains(attrs, "data-ui8kit=\"sheet\"") || hasStaticAttr(attrs, "data-ui8kit-dialog") {
			hasShellMarker = true
			if !containsAttr(attrs, `role="dialog"`) {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT001", reason: "dialog/sheet markup must include role=\"dialog\""})
			}
			if !containsAttr(attrs, `aria-modal="true"`) {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT002", reason: "dialog/sheet markup must include aria-modal=\"true\""})
			}
			if !strings.Contains(attrs, "aria-labelledby=") && !strings.Contains(attrs, "aria-label=") {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT003", reason: "dialog/sheet markup must include aria-label or aria-labelledby"})
			}
			if id := firstValue(idAttrRE, tagText); id != "" {
				dialogIDs[id] = true
			}
		}

		if hasStaticAttr(attrs, "data-ui8kit-dialog-open") {
			hasShellMarker = true
			target := firstValue(targetAttrRE, tagText)
			if target == "" {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT010", reason: "dialog trigger must include data-ui8kit-dialog-target"})
			} else if !ids[target] {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT011", reason: fmt.Sprintf("dialog trigger targets missing id %q", target)})
			} else {
				triggerTargets[target] = true
			}
			if !containsAttr(attrs, `aria-haspopup="dialog"`) {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT012", reason: "dialog trigger must include aria-haspopup=\"dialog\""})
			}
			if !strings.Contains(attrs, "aria-controls=") {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT013", reason: "dialog trigger must include aria-controls"})
			}
		}

		if hasStaticAttr(attrs, "data-ui8kit-dialog-close") {
			if target := firstValue(targetAttrRE, tagText); target != "" {
				closeTargets[target] = true
			}
		}

		if strings.Contains(attrs, `id="ui8kit-theme-toggle"`) {
			if !strings.Contains(attrs, "data-switch-to-dark-label=") {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT020", reason: "theme toggle must include data-switch-to-dark-label"})
			}
			if !strings.Contains(attrs, "data-switch-to-light-label=") {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT021", reason: "theme toggle must include data-switch-to-light-label"})
			}
			if !containsAttr(attrs, `aria-pressed="false"`) {
				out = append(out, violation{file: file, line: line, rule: "LAYOUT022", reason: "theme toggle must include aria-pressed=\"false\" initial state"})
			}
		}
	}

	for id := range dialogIDs {
		if !triggerTargets[id] {
			out = append(out, violation{file: file, line: 1, rule: "LAYOUT032", reason: fmt.Sprintf("dialog/sheet %q has no trigger targeting it", id)})
		}
		if !closeTargets[id] {
			out = append(out, violation{file: file, line: 1, rule: "LAYOUT030", reason: fmt.Sprintf("dialog/sheet %q has no close control targeting it", id)})
		}
	}
	for target := range targets {
		if !ids[target] {
			out = append(out, violation{file: file, line: 1, rule: "LAYOUT031", reason: fmt.Sprintf("dialog target %q does not match any static id", target)})
		}
	}
	if hasShellMarker && !hasMain {
		out = append(out, violation{file: file, line: 1, rule: "LAYOUT040", reason: "custom shell with dialog/sheet markup should include a <main> landmark"})
	}
	if hasShellMarker && !providedContracts["theme.toggle"] && !strings.Contains(content, `id="ui8kit-theme-toggle"`) {
		out = append(out, violation{file: file, line: 1, rule: "LAYOUT041", reason: "custom shell should include a UI8Kit-compatible theme.toggle provider or DOM contract"})
	}

	return out
}

func detectContractProviders(file, content string, registry contractRegistry) (map[string]bool, map[string]map[string]bool, []violation) {
	provided := map[string]bool{}
	targets := map[string]map[string]bool{}
	var out []violation
	importAliases := collectImportAliases(content)

	for contractName, contract := range registry.Contracts {
		for _, provider := range contract.Providers {
			importPath, componentName, ok := splitComponent(provider.Component)
			if !ok {
				continue
			}
			alias, ok := importAliases[importPath]
			if !ok {
				continue
			}

			selector := alias + "." + componentName
			callRE := regexp.MustCompile(`(?s)@` + regexp.QuoteMeta(selector) + `\s*\((.*?)\)`)
			for _, match := range callRE.FindAllStringSubmatchIndex(content, -1) {
				provided[contractName] = true
				callText := content[match[0]:match[1]]
				line := lineAt(content, match[0])

				if provider.TargetField == "" {
					continue
				}
				targetRE := regexp.MustCompile(`\b` + regexp.QuoteMeta(provider.TargetField) + `\s*:\s*"([^"]+)"`)
				target := firstValue(targetRE, callText)
				if target == "" {
					out = append(out, violation{file: file, line: line, rule: "LAYOUT014", reason: fmt.Sprintf("%s provider %s must pass a static %s", contractName, provider.Component, provider.TargetField)})
					continue
				}
				if targets[contractName] == nil {
					targets[contractName] = map[string]bool{}
				}
				targets[contractName][target] = true
			}
		}
	}

	return provided, targets, out
}

func collectImportAliases(content string) map[string]string {
	aliases := map[string]string{}
	for _, match := range importRE.FindAllStringSubmatch(content, -1) {
		alias := strings.TrimSpace(match[1])
		importPath := match[2]
		if !strings.Contains(importPath, "/") {
			continue
		}
		if alias == "" {
			alias = filepath.Base(importPath)
		}
		aliases[importPath] = alias
	}
	return aliases
}

func splitComponent(component string) (string, string, bool) {
	index := strings.LastIndex(component, ".")
	if index <= 0 || index == len(component)-1 {
		return "", "", false
	}
	return component[:index], component[index+1:], true
}

func collectValues(re *regexp.Regexp, content string) map[string]bool {
	values := map[string]bool{}
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		values[match[1]] = true
	}
	return values
}

func firstValue(re *regexp.Regexp, content string) string {
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func containsAttr(attrs, value string) bool {
	return strings.Contains(attrs, value)
}

func hasStaticAttr(attrs, name string) bool {
	for _, field := range strings.Fields(attrs) {
		if field == name || strings.HasPrefix(field, name+"=") {
			return true
		}
	}
	return false
}

func lineAt(content string, index int) int {
	return strings.Count(content[:index], "\n") + 1
}
