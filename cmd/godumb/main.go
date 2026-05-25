package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pecigonzalo/godumb/internal/godumb"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "fmt":
		if err := runFmt(args[1:], stdin, stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "transpile":
		if err := runTranspile(args[1:], stdin, stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "build":
		if err := runBuild(args[1:], stdout, stderr); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "run":
		if err := runRun(args[1:], stdin, stdout, stderr); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "check":
		if err := runCheck(args[1:], stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runFmt(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		writeInPlace bool
		keepComments bool
	)

	fs.BoolVar(&writeInPlace, "w", false, "write result to (source) file instead of stdout")
	fs.BoolVar(&keepComments, "comments", false, "keep comments in output")

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := fs.Args()
	if writeInPlace && len(paths) == 0 {
		return errors.New("-w requires at least one file")
	}
	if !writeInPlace && len(paths) > 1 {
		return errors.New("multiple files without -w are ambiguous")
	}

	opts := godumb.FormatOptions{KeepComments: keepComments}

	if len(paths) == 0 {
		src, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}

		formatted, err := godumb.Format(src, opts)
		if err != nil {
			return err
		}

		_, err = io.WriteString(stdout, formatted)
		return err
	}

	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		formatted, err := godumb.Format(src, opts)
		if err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}

		if writeInPlace {
			if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}
			continue
		}

		if _, err := io.WriteString(stdout, formatted); err != nil {
			return err
		}
	}

	return nil
}

func runTranspile(args []string, stdin io.Reader, stdout io.Writer) error {
	fs := flag.NewFlagSet("transpile", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		writeDerived bool
		outPath      string
	)

	fs.BoolVar(&writeDerived, "w", false, "write each result to a derived .go path")
	fs.StringVar(&outPath, "o", "", "write result to a specific output file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := fs.Args()
	if writeDerived && outPath != "" {
		return errors.New("-w and -o cannot be used together")
	}
	if writeDerived && len(paths) == 0 {
		return errors.New("-w requires at least one file")
	}
	if outPath != "" && len(paths) > 1 {
		return errors.New("-o supports only one input")
	}
	if !writeDerived && outPath == "" && len(paths) > 1 {
		return errors.New("multiple files without -w are ambiguous")
	}

	if len(paths) == 0 {
		src, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}

		goSrc, err := godumb.Transpile(src, godumb.TranspileOptions{})
		if err != nil {
			return err
		}

		if outPath != "" {
			return os.WriteFile(outPath, []byte(goSrc), 0o644)
		}

		_, err = io.WriteString(stdout, goSrc)
		return err
	}

	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		goSrc, err := godumb.Transpile(src, godumb.TranspileOptions{})
		if err != nil {
			return fmt.Errorf("transpile %s: %w", path, err)
		}

		switch {
		case writeDerived:
			dst := derivedGoPath(path)
			if err := os.WriteFile(dst, []byte(goSrc), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", dst, err)
			}
		case outPath != "":
			if err := os.WriteFile(outPath, []byte(goSrc), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", outPath, err)
			}
		default:
			if _, err := io.WriteString(stdout, goSrc); err != nil {
				return err
			}
		}
	}

	return nil
}

func derivedGoPath(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".gdb", ".godumb":
		return strings.TrimSuffix(path, ext) + ".go"
	default:
		return path + ".go"
	}
}

var goBuildRunner = runGoBuild
var binaryRunner = runBinary

func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var outPath string
	fs.StringVar(&outPath, "o", "", "write binary to output path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := fs.Args()
	if len(paths) != 1 {
		return errors.New("build requires exactly one input file")
	}

	tmpGoPath, cleanupGo, err := transpileToTempGo(paths[0])
	if err != nil {
		return err
	}
	defer cleanupGo()

	goArgs := []string{"build"}
	if outPath != "" {
		goArgs = append(goArgs, "-o", outPath)
	}
	goArgs = append(goArgs, tmpGoPath)

	if err := goBuildRunner(goArgs, stdout, stderr); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(goArgs, " "), err)
	}

	return nil
}

func runRun(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := fs.Args()
	if len(paths) == 0 {
		return errors.New("run requires an input file")
	}

	sourcePath := paths[0]
	programArgs := paths[1:]

	tmpGoPath, cleanupGo, err := transpileToTempGo(sourcePath)
	if err != nil {
		return err
	}
	defer cleanupGo()

	tmpBinary, err := os.CreateTemp("", "godumb-run-*")
	if err != nil {
		return fmt.Errorf("create temp binary: %w", err)
	}
	tmpBinaryPath := tmpBinary.Name()
	if err := tmpBinary.Close(); err != nil {
		return fmt.Errorf("close temp binary: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpBinaryPath)
	}()

	goArgs := []string{"build", "-o", tmpBinaryPath, tmpGoPath}
	if err := goBuildRunner(goArgs, stdout, stderr); err != nil {
		return fmt.Errorf("go %s: %w", strings.Join(goArgs, " "), err)
	}

	if err := binaryRunner(tmpBinaryPath, programArgs, stdin, stdout, stderr); err != nil {
		return fmt.Errorf("run binary: %w", err)
	}

	return nil
}

func runCheck(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	paths := fs.Args()
	if len(paths) == 0 {
		return errors.New("check requires at least one input file")
	}

	for _, path := range paths {
		if err := checkFile(path, stdout); err != nil {
			return err
		}
	}

	return nil
}

func checkFile(path string, stdout io.Writer) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	transpiled, err := godumb.TranspileWithSourceMap(src, godumb.TranspileOptions{})
	if err != nil {
		return fmt.Errorf("transpile %s: %w", path, err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path+".gen.go", transpiled.Source, parser.AllErrors)
	if err != nil {
		return mapParserError(path, err, transpiled.Mapper)
	}

	var typeErrors []types.Error
	cfg := &types.Config{
		Importer: importer.Default(),
		Error: func(err error) {
			if typed, ok := err.(types.Error); ok {
				typeErrors = append(typeErrors, typed)
				return
			}
			typeErrors = append(typeErrors, types.Error{Msg: err.Error()})
		},
	}

	_, _ = cfg.Check(path, fset, []*ast.File{file}, nil)
	if len(typeErrors) > 0 {
		return mapTypeErrors(path, typeErrors, fset, transpiled.Mapper)
	}

	_, _ = fmt.Fprintf(stdout, "ok: %s\n", path)
	return nil
}

func mapParserError(path string, err error, mapper godumb.PositionMapper) error {
	var entries []string
	switch parsed := err.(type) {
	case scanner.ErrorList:
		for _, item := range parsed {
			entries = append(entries, formatMappedDiagnostic(path, item.Pos.Line, item.Pos.Column, item.Msg, mapper))
		}
	case *scanner.ErrorList:
		for _, item := range *parsed {
			entries = append(entries, formatMappedDiagnostic(path, item.Pos.Line, item.Pos.Column, item.Msg, mapper))
		}
	default:
		entries = append(entries, fmt.Sprintf("%s: %v", path, err))
	}

	return errors.New(strings.Join(entries, "\n"))
}

func mapTypeErrors(path string, errs []types.Error, fset *token.FileSet, mapper godumb.PositionMapper) error {
	entries := make([]string, 0, len(errs))
	for _, item := range errs {
		pos := fset.Position(item.Pos)
		entries = append(entries, formatMappedDiagnostic(path, pos.Line, pos.Column, item.Msg, mapper))
	}
	return errors.New(strings.Join(entries, "\n"))
}

func formatMappedDiagnostic(path string, generatedLine, generatedColumn int, msg string, mapper godumb.PositionMapper) string {
	if sourceLine, ok := mapper.Map(generatedLine, generatedColumn); ok {
		return fmt.Sprintf("%s:%d: %s (generated %d:%d)", path, sourceLine, msg, generatedLine, generatedColumn)
	}

	if generatedLine > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", path, generatedLine, generatedColumn, msg)
	}
	return fmt.Sprintf("%s: %s", path, msg)
}

func transpileToTempGo(sourcePath string) (string, func(), error) {
	src, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", sourcePath, err)
	}

	goSrc, err := godumb.Transpile(src, godumb.TranspileOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("transpile %s: %w", sourcePath, err)
	}

	dir := filepath.Dir(sourcePath)
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	tmp, err := os.CreateTemp(dir, base+".*.godumb.gen.go")
	if err != nil {
		return "", nil, fmt.Errorf("create temp go file: %w", err)
	}

	tmpPath := tmp.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if _, err := tmp.WriteString(goSrc); err != nil {
		cleanup()
		_ = tmp.Close()
		return "", nil, fmt.Errorf("write temp go file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temp go file: %w", err)
	}

	return tmpPath, cleanup, nil
}

func runGoBuild(args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command("go", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func runBinary(path string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := exec.Command(path, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `GoDumb: line-count-hyperoptimized Go

Usage:
  godumb fmt [flags] [path]
  godumb transpile [flags] [path]
  godumb build [flags] <path.gdb>
  godumb run <path.gdb> [program args...]
  godumb check <path.gdb> [...more]

Commands:
  fmt         Format Go code into GoDumb style (one token per line)
  transpile   Convert GoDumb source back to canonical Go
  build       Transpile and compile GoDumb source into a binary
  run         Transpile, compile, and execute GoDumb source
  check       Validate GoDumb source and map diagnostics to .gdb lines

Flags (fmt):
  -w            write result to file
  -comments     keep comments in output

Flags (transpile):
  -w            write each input to a derived .go file
  -o <path>     write result to a specific output file

Flags (build):
  -o <path>     write binary to output path
`)
}
