// Command coverage-gate enforces per-package coverage thresholds for
// the framework. It reads a Go cover profile (default coverage.out)
// and fails the build when any tracked package drops below the
// declared minimum.
//
// Usage (from repo root):
//
//	go test -covermode=atomic -coverprofile=coverage.out ./pkg/...
//	go run ./scripts/coverage-gate
//
//	# Custom profile path:
//	go run ./scripts/coverage-gate -profile=path/to/coverage.out
//
//	# Override the global threshold (per-package overrides still apply):
//	go run ./scripts/coverage-gate -default=85
//
// Exit code 0 means every tracked package meets its threshold. Exit
// code 1 prints a sorted report (worst-violator first) and the global
// summary, then fails. Untracked packages are reported but never
// fail the gate.
//
// Thresholds are stored in this file (see thresholds below) so they
// live next to the code that enforces them and mutate via PR review.
// The same numbers are mirrored in CHANGELOG / roadmap.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// thresholds maps a Go import path to the minimum acceptable
// statement-coverage percentage. Unlisted packages fall back to
// defaultThreshold; security-critical packages get tighter numbers.
//
// IMPORTANT: this map is the single source of truth for the gate.
// Keep it in sync with docs/SECURITY.md, CHANGELOG, and
// .project/roadmap-framework.md when raising or lowering a target.
var thresholds = map[string]float64{
	// Security-critical: HMAC sessions, OIDC, security middleware.
	"github.com/fastygo/framework/pkg/auth":         90.0,
	"github.com/fastygo/framework/pkg/web/security": 90.0,

	// Core domain primitives and the CQRS dispatcher.
	"github.com/fastygo/framework/pkg/core":                80.0,
	"github.com/fastygo/framework/pkg/core/cqrs":           80.0,
	"github.com/fastygo/framework/pkg/core/cqrs/behaviors": 80.0,

	// Composition root: Builder, App lifecycle, workers.
	"github.com/fastygo/framework/pkg/app": 85.0,

	// Infrastructure with full test suites.
	"github.com/fastygo/framework/pkg/cache":          90.0,
	"github.com/fastygo/framework/pkg/observability":  80.0,
	"github.com/fastygo/framework/pkg/web/health":     90.0,
	"github.com/fastygo/framework/pkg/web/instant":    90.0,
	"github.com/fastygo/framework/pkg/web/metrics":    80.0,
	"github.com/fastygo/framework/pkg/web/middleware": 80.0,
	"github.com/fastygo/framework/pkg/web/view":       80.0,

	// Lower bar — rendering and locale loaders are exercised mostly
	// via integration. Tracked so regressions still get noticed.
	"github.com/fastygo/framework/pkg/content-markdown": 65.0,
	"github.com/fastygo/framework/pkg/web/i18n":         60.0,
	"github.com/fastygo/framework/pkg/web/locale":       70.0,
}

// defaultThreshold is used for any package not listed in thresholds.
// Untracked packages still report their coverage but never fail.
const defaultThreshold = 0.0

func main() {
	profilePath := flag.String("profile", "coverage.out", "path to the cover profile")
	override := flag.Float64("default", defaultThreshold, "global threshold for untracked packages (use to tighten the floor)")
	flag.Parse()

	if _, err := os.Stat(*profilePath); err != nil {
		fmt.Fprintf(os.Stderr, "coverage-gate: profile not found: %s\n", *profilePath)
		fmt.Fprintln(os.Stderr, "Run: go test -covermode=atomic -coverprofile="+*profilePath+" ./pkg/...")
		os.Exit(2)
	}

	pkgCoverage, err := parseProfile(*profilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "coverage-gate: parse profile: %v\n", err)
		os.Exit(2)
	}

	report, failed := evaluate(pkgCoverage, *override)
	printReport(os.Stdout, report)
	if failed {
		fmt.Fprintln(os.Stderr, "\ncoverage-gate: FAIL — one or more packages below threshold")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "\ncoverage-gate: OK — all tracked packages at or above threshold")
}

// pkgRow is one row in the report: import path + coverage + threshold.
type pkgRow struct {
	pkg       string
	coverage  float64
	threshold float64
	ok        bool
	tracked   bool
}

func evaluate(coverage map[string]float64, override float64) ([]pkgRow, bool) {
	rows := make([]pkgRow, 0, len(coverage))
	failed := false

	for pkg, cov := range coverage {
		threshold, tracked := thresholds[pkg]
		if !tracked {
			threshold = override
		}
		ok := cov+1e-9 >= threshold
		if tracked && !ok {
			failed = true
		}
		rows = append(rows, pkgRow{
			pkg:       pkg,
			coverage:  cov,
			threshold: threshold,
			ok:        ok,
			tracked:   tracked,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		// Failures first (worst delta on top), then alphabetical.
		if rows[i].ok != rows[j].ok {
			return !rows[i].ok
		}
		di := rows[i].coverage - rows[i].threshold
		dj := rows[j].coverage - rows[j].threshold
		if di != dj {
			return di < dj
		}
		return rows[i].pkg < rows[j].pkg
	})

	return rows, failed
}

func printReport(out *os.File, rows []pkgRow) {
	fmt.Fprintln(out, "coverage-gate report")
	fmt.Fprintln(out, "--------------------")
	fmt.Fprintf(out, "%-8s %-58s %8s %8s\n", "STATUS", "PACKAGE", "COV", "MIN")
	for _, r := range rows {
		status := "OK"
		switch {
		case !r.tracked:
			status = "untracked"
		case !r.ok:
			status = "FAIL"
		}
		fmt.Fprintf(out, "%-8s %-58s %7.1f%% %7.1f%%\n",
			status, r.pkg, r.coverage, r.threshold)
	}
}

// parseProfile reads a Go cover profile and returns per-package
// statement coverage as a percentage (0..100). The profile lines
// after the mode header have the form
//
//	import/path/to/file.go:startLine.col,endLine.col numStatements count
//
// We aggregate per package (= dirname of the file path).
//
// We deliberately re-implement the parser instead of shelling out to
// `go tool cover -func` so the gate has zero external dependencies
// and works on Windows without a POSIX shell.
func parseProfile(path string) (map[string]float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	type counter struct {
		stmts   int
		covered int
	}
	perPkg := map[string]*counter{}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 && strings.HasPrefix(line, "mode:") {
			continue
		}

		// Split into "<filepath:range> <numStmt> <count>".
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("line %d: expected 3 fields, got %d (%q)", i+1, len(fields), line)
		}

		pathRange := fields[0]
		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: numStatements: %w", i+1, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: count: %w", i+1, err)
		}

		// pathRange = "import/path/to/file.go:start.col,end.col"
		colon := strings.IndexByte(pathRange, ':')
		if colon < 0 {
			return nil, fmt.Errorf("line %d: missing ':' in %q", i+1, pathRange)
		}
		filePath := pathRange[:colon]
		pkg := filepath.ToSlash(filepath.Dir(filePath))

		c, ok := perPkg[pkg]
		if !ok {
			c = &counter{}
			perPkg[pkg] = c
		}
		c.stmts += numStmt
		if count > 0 {
			c.covered += numStmt
		}
	}

	out := make(map[string]float64, len(perPkg))
	for pkg, c := range perPkg {
		if c.stmts == 0 {
			out[pkg] = 0
			continue
		}
		out[pkg] = float64(c.covered) / float64(c.stmts) * 100
	}
	return out, nil
}

// silence "imported and not used" if exec is dropped during edits.
var _ = exec.Command
