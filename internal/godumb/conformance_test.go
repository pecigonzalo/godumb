package godumb_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/pecigonzalo/godumb/internal/godumb"
)

func TestTranspileConformance_GoByExampleCorpus(t *testing.T) {
	root := findGoByExampleDir(t)

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	var slugs []string
	for _, entry := range entries {
		if entry.IsDir() {
			slugs = append(slugs, entry.Name())
		}
	}
	sort.Strings(slugs)

	if len(slugs) == 0 {
		t.Fatalf("expected Go by Example samples under %s", root)
	}

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			t.Parallel()

			gdbPath := filepath.Join(root, slug, "main.gdb")

			gdbSrc, err := os.ReadFile(gdbPath)
			if err != nil {
				t.Fatalf("read %s: %v", gdbPath, err)
			}

			transpiled, err := godumb.Transpile(gdbSrc, godumb.TranspileOptions{})
			if err != nil {
				t.Fatalf("transpile failed: %v", err)
			}

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, slug+".go", transpiled, parser.AllErrors)
			if err != nil {
				t.Fatalf("transpiled output should parse: %v\n%s", err, transpiled)
			}

			cfg := &types.Config{Importer: importer.Default()}
			if _, err := cfg.Check(slug, fset, []*ast.File{file}, nil); err != nil {
				t.Fatalf("transpiled output should type-check: %v\n%s", err, transpiled)
			}
		})
	}
}

func findGoByExampleDir(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "examples", "gobyexample"),
		filepath.Join("examples", "gobyexample"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	t.Fatalf("could not find examples/gobyexample directory")
	return ""
}
