package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pecigonzalo/godumb/internal/godumb"
)

const goByExampleRawBase = "https://raw.githubusercontent.com/mmcgrana/gobyexample/master/examples"

var defaultExamples = []string{
	"hello-world",
	"values",
	"variables",
	"constants",
	"for",
	"if-else",
	"switch",
	"arrays",
	"slices",
	"maps",
	"range-over-built-in-types",
	"functions",
	"multiple-return-values",
	"variadic-functions",
	"closures",
}

var exampleAliases = map[string]string{
	"range": "range-over-built-in-types",
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("godumb-examples", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		outDir  string
		onlyArg string
		timeout time.Duration
		clean   bool
	)

	fs.StringVar(&outDir, "out", "examples/gobyexample", "output directory")
	fs.StringVar(&onlyArg, "only", "", "comma-separated example slugs to fetch")
	fs.DurationVar(&timeout, "timeout", 20*time.Second, "HTTP timeout")
	fs.BoolVar(&clean, "clean", false, "remove output directory before syncing")

	if err := fs.Parse(args); err != nil {
		return err
	}

	selected, err := selectExamples(onlyArg)
	if err != nil {
		return err
	}

	if clean {
		if err := os.RemoveAll(outDir); err != nil {
			return fmt.Errorf("clean %s: %w", outDir, err)
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}

	client := &http.Client{Timeout: timeout}

	for _, slug := range selected {
		goURL := fmt.Sprintf("%s/%s/%s.go", goByExampleRawBase, slug, slug)
		goSrc, err := fetchText(client, goURL)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", slug, err)
		}

		gdbSrc, err := godumb.Format([]byte(goSrc), godumb.FormatOptions{})
		if err != nil {
			return fmt.Errorf("format %s: %w", slug, err)
		}

		exampleDir := filepath.Join(outDir, slug)
		if err := os.MkdirAll(exampleDir, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", exampleDir, err)
		}

		goPath := filepath.Join(exampleDir, "main.go")
		gdbPath := filepath.Join(exampleDir, "main.gdb")

		if err := os.WriteFile(goPath, []byte(ensureTrailingNewline(goSrc)), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", goPath, err)
		}
		if err := os.WriteFile(gdbPath, []byte(gdbSrc), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", gdbPath, err)
		}

		_, _ = fmt.Fprintf(stdout, "synced %s\n", slug)
	}

	_, _ = fmt.Fprintf(stderr, "done: %d examples in %s\n", len(selected), outDir)
	return nil
}

func selectExamples(onlyArg string) ([]string, error) {
	if strings.TrimSpace(onlyArg) == "" {
		selected := append([]string(nil), defaultExamples...)
		sort.Strings(selected)
		return selected, nil
	}

	allowed := make(map[string]struct{}, len(defaultExamples))
	for _, slug := range defaultExamples {
		allowed[slug] = struct{}{}
	}

	parts := strings.Split(onlyArg, ",")
	seen := map[string]struct{}{}
	selected := make([]string, 0, len(parts))
	for _, part := range parts {
		slug := strings.TrimSpace(part)
		if slug == "" {
			continue
		}
		if canonical, ok := exampleAliases[slug]; ok {
			slug = canonical
		}
		if _, ok := allowed[slug]; !ok {
			return nil, fmt.Errorf("unknown example slug: %s", slug)
		}
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}
		selected = append(selected, slug)
	}

	if len(selected) == 0 {
		return nil, errors.New("no examples selected")
	}

	sort.Strings(selected)
	return selected, nil
}

func fetchText(client *http.Client, url string) (string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func ensureTrailingNewline(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}
