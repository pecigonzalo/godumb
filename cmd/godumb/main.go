package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

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

func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, `GoDumb: line-count-hyperoptimized Go

Usage:
  godumb fmt [flags] [path]

Commands:
  fmt       Format Go code into GoDumb style (one token per line)

Flags (fmt):
  -w            write result to file
  -comments     keep comments in output
`)
}
